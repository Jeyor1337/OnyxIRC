This is an irc client/server project.

Its server uses golang as the back end.

Its encryption method is rsa/aes256.

Hash calculation uses sha256.

Have a perfect decentralized management system/administrator command system/dummy prevention system/Configuration file system.

Clients can be implemented in multiple languages, and java is used as an example client language here.

Communication logic：

1. After collecting the username/password/ip address, the client sends it to the server with sha256 encryption(excluding ip).

2.After receiving the client request, the server compares the username/password hash value in the database, and at the same time checks whether the current requested ip is the same as the ip accessed last time.

3.
 - (1) If the ip addresses are the same, login is allowed directly.
 - (2) If the current ip is different from the last ip, add one to the ip suspicion.
 - (3) Prohibit users with ip suspicion greater than 3 from logging in.
 
4.Rsa/aes is used for data transmission.

5.Intelligent multithreading

6.You also need to design mysql architecture and statements yourself.