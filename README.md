# FileTransfer
## Description
Local webserver to allow transfer files/pictures between devices(including phone).
## Installation
- Download repository
- Install [Golang](https://go.dev/).
- Run ```go build``` from directory.
- Run the created Executable/Binary.
## Config
- ```<user>``` is set per web session
- Saves files in the directory ```<userPath><user><subfolder>```
- Requires directory to already exist
- Default path is: ```C:\\Users\\<user>\\```
- ```<port>``` is the port the webserver runs on
