# Change Log

## V1.1.0 <small> Mar 2022 </small>
### Features
1. dai
    1. model evaluation: support model evaluation and live evaluation through random split, cross validation or LOO
    2. self-computing: support two computing modes for executors, one is to compute using others' data, one is to compute using their own data
### Improvements
1. crypto
    1. improve security of pairing-based PoRH: upgrade algorithm to avoid mod-N attack, which enables storage nodes to pass challenges using only slice data mod N

## V1.0.0 <small> Jan 2022 </small>
This is the first public release of PaddleDTX.

### Features
1. dai
    1. vertical federated learning(linear regression and logistic regression)
    2. training/prediction task management
    3. p2p
    4. requester cli
    5. support xuperchain

2. xdb
    1. file upload and download
    2. file slicing and encryption
    3. file copies making and distribution
    4. proof of replication holding(pairing/merkle)
    5. health detection
    6. file migration
    7. resource access control
    8. support xuperchain and fabric

3. crypto
    1.  distributed implementation of linear regression
    2. distributed implementation of logistic regression
    3. additive homomorphic encryption(Paillier)
    4. private set intersection
    5. oblivious transfer
    6. secret sharing
    7. proof of replication holding

<br>
