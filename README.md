# Go tftp server and client

Tftp server and client that implement:
* [RFC 1350](https://datatracker.ietf.org/doc/html/rfc1350) - The TFTP Protocol (Revision 2)

### Client Usage
````bash
$ go run cmd/client/main.go
tftp> help
Commands:
        connect <host> <port>
        get <file>
        put <file>
        timeout <integer>
        trace
        quit
````

### Config
| Name                     | Use-Case                                                                          | Default value |
|--------------------------|-----------------------------------------------------------------------------------|---------------|
| `TFTP_PORT`              | Tftp server port                                                                  | 69            |
| `TFTP_LOG_LEVEL`         | Log level                                                                         | debug         |
| `TFTP_READ_TIMEOUT`      | Timeout while reading tftp request in seconds                                     | 5             |
| `TFTP_WRITE_TIMEOUT`     | Timeout while writing tftp request in seconds                                     | 5             |
| `TFTP_NUM_TRIES`         | Number of times that a read/write request should be executed if one of them fails | 5             |
| `TFTP_BASE_DIR`          | Tftp folder, where file can stored and pulled from                                | ~./tftp       |
| `TFTP_TRACE`             | Log each sent/received udp packet                                                 | false         |

### Example get request
````bash
tftp> get <file>
received block#=1, received #bytes=512
received block#=2, received #bytes=512
received block#=3, received #bytes=512
received block#=4, received #bytes=512
received block#=5, received #bytes=512
....
received block#=33, received #bytes=512
received block#=34, received #bytes=512
received block#=35, received #bytes=512
received block#=36, received #bytes=289
received 36 blocks, received 18209 bytes
tftp>
````

### Server usage
````bash
$ go run cmd/server/main.go
2024-03-01T19:36:40.815+0100    INFO    server/main.go:32       listening on port 69
````

### Example logs when tftp server is serving a file
````bash
sent block#=1, sent #bytes=512
sent block#=2, sent #bytes=512
sent block#=3, sent #bytes=512
sent block#=4, sent #bytes=512
sent block#=5, sent #bytes=512
...
sent block#=33, sent #bytes=512
sent block#=34, sent #bytes=512
sent block#=35, sent #bytes=512
sent block#=36, sent #bytes=289
sent 37 blocks, sent 18209 bytes
````