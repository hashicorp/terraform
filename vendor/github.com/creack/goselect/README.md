# go-select

select(2) implementation in Go

## Supported platforms

|               | 386 | amd64 | arm | arm64 |
|---------------|-----|-------|-----|-------|
| **linux**     | yes | yes   | yes | yes   |
| **darwin**    | yes | yes   | n/a | ??    |
| **freebsd**   | yes | yes   | yes | ??    |
| **openbsd**   | yes | yes   | yes | ??    |
| **netbsd**    | yes | yes   | yes | ??    |
| **dragonfly** | n/a | yes   | n/a | ??    |
| **solaris**   | n/a | no    | n/a | ??    |
| **plan9**     | no  | no    | n/a | ??    |
| **windows**   | yes | yes   | n/a | ??    |
| **android**   | n/a | n/a   | no  | ??    |

*n/a: platform not supported by Go

Go on `plan9` and `solaris` do not implement `syscall.Select` not `syscall.SYS_SELECT`.

## Cross compile

Using davecheney's https://github.com/davecheney/golang-crosscompile

```
export PLATFORMS="darwin/386 darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm windows/386 windows/amd64 openbsd/386 openbsd/amd64 netbsd/386 netbsd/amd64 dragonfly/amd64 plan9/386 plan9/amd64 solaris/amd64"
```
