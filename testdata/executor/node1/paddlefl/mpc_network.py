import paddle.fluid as fluid
import paddle_fl.mpc as pfl_mpc

class PaddleDNN(object):
    def input_data(self, batch_size, part0_vec_size, part1_vec_size, part2_vec_size, label_vec_size):
        part0 = pfl_mpc.data(name='part0', shape=[batch_size, part0_vec_size], dtype='int64')
        part1 = pfl_mpc.data(name='part1', shape=[batch_size, part1_vec_size], dtype='int64')
        part2 = pfl_mpc.data(name='part2', shape=[batch_size, part2_vec_size], dtype='int64')
        label = pfl_mpc.data(name='label', shape=[batch_size, label_vec_size], dtype='int64')

        inputs = [part0] + [part1] + [part2] + [label]
        return inputs

    def net(self, inputs):

        concat_feats = fluid.layers.concat(input=inputs[:-1], axis=-1)
        l1 = pfl_mpc.layers.fc(input=concat_feats, size=30, act='relu')
        l2 = pfl_mpc.layers.fc(input=l1, size=15, act='relu')
        y_pre = pfl_mpc.layers.fc(input=l2, size=1)
        cost = pfl_mpc.layers.square_error_cost(input=y_pre, label=inputs[-1])
        avg_loss = pfl_mpc.layers.mean(cost)
        return avg_loss, y_pre

        '''

        l1 = self.fc('l1', concat_feats, layers[0], 'relu')
        l2 = self.fc('l2', l1, layers[1], 'relu')
        l3 = self.fc('l3', l2, layers[2], 'relu')
        l4 = self.fc('l4', l3, output_size, None)
        cost, softmax = pfl_mpc.layers.softmax_with_cross_entropy(logits=l4,
                                                                  label=inputs[-1],
                                                                  soft_label=True,
                                                                  use_relu=True,
                                                                  use_long_div=False,
                                                                  return_softmax=True)
        avg_cost = pfl_mpc.layers.mean(cost)
        return avg_cost, l3
        '''