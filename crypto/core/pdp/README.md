# Proof of Data Possession
Proof of data possession is a protocol that allows the client to verify the data file stored on a remote server, the availability of the original version. 

The client and server exchange messages according to the model Request - Answer.

Protocol PDP consists of four treatments: pre-process, inquiry, confirmation, check. 
- The client C (data owner) preprocesses the file, generating a small piece of metadata that is stored locally, transmits the file to the server S, and may delete its local copy. 
- The server stores the file and responds to challenges issued by the client. 
- To verify ownership of a file, the client sends a random challenge to the server to check for evidence of the specified file. 
- The server generates a proof in response. This calculation requires the possession of the original data and the data of the current request, to avoid repeated attacks. 
- Upon receipt, the client compares the evidence with a locally stored file metadata.

## protocol
http://wiki.baidu.com/display/BlockChainLaboratory/tips

## Reference
Proof of data possession: http://cryptowiki.net/index.php?title=Proof_of_data_possession
