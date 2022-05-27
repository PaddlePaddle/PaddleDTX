import argparse
import paddle_fl.mpc as pfl_mpc
import numpy as np
import paddle.fluid as fluid
import logging
import mpc_network
import os
import time
from paddle_fl.mpc.data_utils.data_utils import get_datautils



logging.basicConfig(format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger('fluid')
logger.setLevel(logging.INFO)

def parse_args():
    parser = argparse.ArgumentParser(description='process data using aby3')
    parser.add_argument('--func', help='func to train')
    parser.add_argument('--samples', help='')
    parser.add_argument('--label', help='')
    parser.add_argument('--parts', help='part\'s grpc address')
    parser.add_argument('--role',type=int,help='role in parts')
    parser.add_argument('--batch_size',type=int, default=1)
    parser.add_argument('--parts_size', help='')
    parser.add_argument('--model_dir', type=str, default='./model_dir/', help='model_dir')
    parser.add_argument('--batch_num', type=int, default=1, help='batch_num')
    parser.add_argument('--base_lr', type=float, default=0.001, help='base_lr')
    parser.add_argument('--use_gpu', type=int, default=0, help='whether using gpu')
    parser.add_argument('--output_size', type=int, default=1, help='output_size')
    parser.add_argument('--output_file', help='output_file')
    parser.add_argument('--epochs', type=int, default=5, help='epochs')

    args = parser.parse_args()
    return args

args = parse_args()
aby3 = get_datautils("aby3")

def train(args):
    servers = ';'.join(args.parts.split(','))
    pfl_mpc.init('aby3', int(args.role), 'localhost', '127.0.0.1', 6379, servers, "grpc")
    dnn_model = mpc_network.PaddleDNN()
    part_size = args.parts_size.split(',')
    inputs = dnn_model.input_data(args.batch_size, int(part_size[0]),
                                      int(part_size[1]),
                                      int(part_size[2]),
                                      args.output_size)
    loss, l3 = dnn_model.net(inputs)
    lr = args.base_lr
    sgd = pfl_mpc.optimizer.SGD(learning_rate=lr)
    sgd.minimize(loss)

    place = fluid.CUDAPlace(0) if args.use_gpu else fluid.CPUPlace()
    exe = fluid.Executor(place)
    exe.run(fluid.default_startup_program())


    # ********************
    # prepare data
    logger.info('Prepare data...')

    ## 读文件名
    file_name = args.samples.split(',')
    part0 = file_name[0]
    part1 = file_name[1]
    part2 = file_name[2]
    label = args.label

    part0_vecs = []
    part1_vecs = []
    part2_vecs = []
    labels = []

    part0_reader = read_share(file=part0, shape=(args.batch_size, int(part_size[0])))
    for vec in part0_reader():
        part0_vecs.append(vec)
    part1_reader = read_share(file=part1, shape=(args.batch_size, int(part_size[1])))
    for vec in part1_reader():
        part1_vecs.append(vec)
    part2_reader = read_share(file=part2, shape=(args.batch_size, int(part_size[2])))
    for vec in part2_reader():
        part2_vecs.append(vec)
    labels_reader = read_share(file=label, shape=(args.batch_size, 1))
    for vec in labels_reader():
        labels.append(vec)


    # ********************
    # train
    logger.info('Start training...')
    begin = time.time()
    for epoch in range(args.epochs):
        for i in range(args.batch_num):
            loss_data = exe.run(fluid.default_main_program(),
                                feed={'part0': part0_vecs[i],
                                      'part1': part1_vecs[i],
                                      'part2': part2_vecs[i],
                                      'label': np.array(labels[i])},
                                return_numpy=True,
                                fetch_list=[loss.name])
            if i % 100 == 0:
                end = time.time()
                logger.info('Paddle training of epoch_id: {}, batch_id: {}, batch_time: {:.5f}s'.format(epoch, i, end-begin))

        # save model
        logger.info('save mpc model...')
        cur_model_dir = os.path.join(args.model_dir, 'mpc_model', 'epoch_' + str(epoch + 1),'checkpoint', 'party_{}'.format(args.role))
        feed_var_names = ['part0', 'part1', 'part2']
        fetch_vars = [l3]
        fluid.io.save_inference_model(cur_model_dir, feed_var_names, fetch_vars, exe)

        end = time.time()
        logger.info('MPC training of epoch: {}, cost_time: {:.5f}s'.format(epoch, end - begin))
    logger.info('End training.')



def read_share(file, shape):
    """
    prepare share reader
    """
    shape = (2, ) + shape
    share_size = np.prod(shape) * 8  # size of int64 in bytes
    def reader():
        with open(file, 'rb') as part_file:
            share = part_file.read(share_size)
            while share:
                yield np.frombuffer(share, dtype=np.int64).reshape(shape)
                share = part_file.read(share_size)
    return reader


def infer(args):
    output_file = args.output_file
    batch_num = args.batch_num
#    output_file = args.output_file + 'out' + '.part{}'.format(args.role)

    logger.info('Start inferring...')
    cur_model_path = os.path.join(args.model_dir, 'mpc_model', 'epoch_' + str(args.epochs),'checkpoint', 'party_{}'.format(args.role))
    print(cur_model_path)
    begin = time.time()
    place = fluid.CUDAPlace(0) if args.use_gpu else fluid.CPUPlace()
    exe = fluid.Executor(place)
    servers = ';'.join(args.parts.split(','))
    part_size = args.parts_size.split(',')
    with fluid.scope_guard(fluid.Scope()):
        pfl_mpc.init('aby3', int(args.role), 'localhost', '127.0.0.1', 6379, servers, "grpc")
        infer_program, feed_target_names, fetch_vars = aby3.load_mpc_model(exe=exe,
                                                                           mpc_model_dir=cur_model_path,
                                                                           mpc_model_filename='__model__',
                                                                           inference=True)
        ## 读文件名
        file_name = args.samples.split(',')
        part0 = file_name[0]
        part1 = file_name[1]
        part2 = file_name[2]

        part0_vecs = []
        part1_vecs = []
        part2_vecs = []

        logger.info('filenem: {}, batch_size: {}, part_size: {}'.format(part0, args.batch_size, int(part_size[0])))

        part0_reader = read_share(file=part0, shape=(args.batch_size, int(part_size[0])))
        for vec in part0_reader():
            part0_vecs.append(vec)
        part1_reader = read_share(file=part1, shape=(args.batch_size, int(part_size[1])))
        for vec in part1_reader():
            part1_vecs.append(vec)
        part2_reader = read_share(file=part2, shape=(args.batch_size, int(part_size[2])))
        for vec in part2_reader():
            part2_vecs.append(vec)


        for i in range(batch_num):
            l3 = exe.run(infer_program,feed={
                'part0': part0_vecs[i],
                'part1': part1_vecs[i],
                'part2': part2_vecs[i],
            },
                         return_numpy=True,
                         fetch_list=fetch_vars)
            with open(output_file, 'ab+') as f:
                f.write(np.array(l3[0]).tostring())
    logger.info('End inferring.')

if __name__ == '__main__':
    if args.func == 'train':
        train(args)
    elif args.func == 'infer':
        infer(args)
