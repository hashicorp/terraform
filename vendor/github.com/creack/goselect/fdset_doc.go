package goselect

/**
From: XCode's MacOSX10.10.sdk/System/Library/Frameworks/Kernel.framework/Versions/A/Headers/sys/select.h
--
// darwin/amd64 / 386
sizeof(__int32_t) == 4
--

typedef __int32_t __fd_mask;

#define	FD_SETSIZE      1024
#define __NFDBITS	(sizeof(__fd_mask) * 8)
#define	__howmany(x, y)	((((x) % (y)) == 0) ? ((x) / (y)) : (((x) / (y)) + 1))

typedef	struct  fd_set {
    __fd_mask   fds_bits[__howmany(__FD_SETSIZE, __NFDBITS)];
}               fd_set;

#define __FD_MASK(n)   ((__fd_mask)1 << ((n) % __NFDBITS))
#define	FD_SET(n, p)    ((p)->fds_bits[(n)/__NFDBITS] |=  __FD_MASK(n))
#define	FD_CLR(n, p)    ((p)->fds_bits[(n)/__NFDBITS] &= ~__FD_MASK(n))
#define FD_ISSET(n, p) (((p)->fds_bits[(n)/__NFDBITS] &   __FD_MASK(n)) != 0)
*/

/**
From: /usr/include/i386-linux-gnu/sys/select.h
--
// linux/i686
sizeof(long int) == 4
--

typedef long int __fd_mask;

#define FD_SETSIZE      1024
#define __NFDBITS       (sizeof(__fd_mask) * 8)


typedef struct  fd_set {
    __fd_mask   fds_bits[__FD_SETSIZE / __NFDBITS];
}               fd_set;

#define __FD_MASK(n)   ((__fd_mask)1 << ((n) % __NFDBITS))
#define FD_SET(n, p)    ((p)->fds_bits[(n)/__NFDBITS] |=  __FD_MASK(n))
#define FD_CLR(n, p)    ((p)->fds_bits[(n)/__NFDBITS] &= ~__FD_MASK(n))
#define FD_ISSET(n, p) (((p)->fds_bits[(n)/__NFDBITS] &   __FD_MASK(n)) != 0)
*/

/**
From: /usr/include/x86_64-linux-gnu/sys/select.h
--
// linux/amd64
sizeof(long int) == 8
--

typedef long int __fd_mask;

#define FD_SETSIZE      1024
#define __NFDBITS       (sizeof(__fd_mask) * 8)


typedef struct  fd_set {
    __fd_mask   fds_bits[__FD_SETSIZE / __NFDBITS];
}               fd_set;

#define __FD_MASK(n)   ((__fd_mask)1 << ((n) % __NFDBITS))
#define FD_SET(n, p)    ((p)->fds_bits[(n)/__NFDBITS] |=  __FD_MASK(n))
#define FD_CLR(n, p)    ((p)->fds_bits[(n)/__NFDBITS] &= ~__FD_MASK(n))
#define FD_ISSET(n, p) (((p)->fds_bits[(n)/__NFDBITS] &   __FD_MASK(n)) != 0)
*/

/**
From: /usr/include/sys/select.h
--
// freebsd/amd64
sizeof(unsigned long) == 8
--

typedef unsigned long   __fd_mask;

#define FD_SETSIZE      1024U
#define __NFDBITS       (sizeof(__fd_mask) * 8)
#define _howmany(x, y)  (((x) + ((y) - 1)) / (y))

typedef struct  fd_set {
    __fd_mask   fds_bits[_howmany(FD_SETSIZE, __NFDBITS)];
}               fd_set;

#define __FD_MASK(n)   ((__fd_mask)1 << ((n) % __NFDBITS))
#define FD_SET(n, p)    ((p)->fds_bits[(n)/__NFDBITS] |=  __FD_MASK(n))
#define FD_CLR(n, p)    ((p)->fds_bits[(n)/__NFDBITS] &= ~__FD_MASK(n))
#define FD_ISSET(n, p) (((p)->fds_bits[(n)/__NFDBITS] &   __FD_MASK(n)) != 0)
*/
