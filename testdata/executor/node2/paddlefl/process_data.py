import numpy as np
import argparse
import six
from paddle_fl.mpc.data_utils.data_utils import get_datautils
import paddle
import pandas as pd


def parse_args():
    parser = argparse.ArgumentParser(description='process data using aby3')
    parser.add_argument('--func', help='func to process datas')
    parser.add_argument('--input', help='file')
    parser.add_argument('--out', help='')
    parser.add_argument('--batch_size', type=int, default=1, help='batch_size')
#    parser.add_argument('--output_size', type=int, default=1, help='output_size')
    parser.add_argument('--path', help='')
    args = parser.parse_args()
    return args

args = parse_args()
aby3 = get_datautils("aby3")


def encrypt_data(input_file, out_files):
    file_names = out_files.split(',')
    if len(file_names) != 3:
        return
    vec = np.loadtxt(input_file, delimiter=',',dtype='float',skiprows=1)
    for v in vec:
        shares = aby3.make_shares(v)
        with open(file_names[0], 'ab') as file0, \
                open(file_names[1], 'ab') as file1, \
                open(file_names[2], 'ab') as file2:
            files = [file0, file1, file2]
            for idx in six.moves.range(0, 3):
                share = aby3.get_shares(shares, idx)
                files[idx].write(share.tostring())

def decrypt_data(path, out_file, shape):
    part_readers = []
    for id in six.moves.range(3):
        part_readers.append(
            aby3.load_shares(
                path, id=id, shape=shape))
    aby3_share_reader = paddle.reader.compose(part_readers[0], part_readers[1],part_readers[2])
    for instance in aby3_share_reader():
        p = aby3.reconstruct(np.array(instance))
        tmp = pd.DataFrame(p)
        tmp.to_csv(out_file, mode='a', index=False, header=0)



if __name__ == '__main__':
    if args.func == 'encrypt_data':
        encrypt_data(args.input, args.out)
    elif args.func == 'decrypt_data':
        decrypt_data(args.path, args.out, (int(args.batch_size),))




