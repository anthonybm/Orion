#ifndef _UTMPX_H_
#define _UTMPX_H_

#include <_types.h>
#include <sys/time.h>
#include <sys/cdefs.h>
#include <Availability.h>
#include <sys/_types/_pid_t.h>

#if !defined(_POSIX_C_SOURCE) || defined(_DARWIN_C_SOURCE)
#include <sys/_types/_uid_t.h>
#endif /* !_POSIX_C_SOURCE || DARWIN_C_SOURCE */

#define _PATH_UTMPX "/var/run/utmpx"

#if !defined(_POSIX_C_SOURCE) || defined(_DARWIN_C_SOURCE)
#define UTMPX_FILE _PATH_UTMPX
#endif /* !_POSIX_C_SOURCE || _DARWIN_C_SOURCE */

#define _UTX_USERSIZE	256	/* matches MAXLOGNAME */
#define _UTX_LINESIZE	32
#define	_UTX_IDSIZE	    4
#define _UTX_HOSTSIZE	256

#define EMPTY		    0
#define RUN_LVL		    1
#define BOOT_TIME	    2
#define OLD_TIME	    3
#define NEW_TIME	    4
#define INIT_PROCESS	5
#define LOGIN_PROCESS	6
#define USER_PROCESS	7
#define DEAD_PROCESS	8

#if !defined(_POSIX_C_SOURCE) || defined(_DARWIN_C_SOURCE)
#define ACCOUNTING	    9
#define SIGNATURE	    10
#define SHUTDOWN_TIME	11

#define UTMPX_AUTOFILL_MASK			        0x8000
#define UTMPX_DEAD_IF_CORRESPONDING_MASK	0x4000

/* notify(3) change notification name */
#define UTMPX_CHANGE_NOTIFICATION		"com.apple.system.utmpx"
#endif /* !_POSIX_C_SOURCE || _DARWIN_C_SOURCE */

/*
 * The following structure describes the fields of the utmpx entries
 * stored in _PATH_UTMPX. This is not the format the
 * entries are stored in the files, and application should only access
 * entries using routines described in getutxent(3).
 */

#ifdef _UTMPX_COMPAT
#define ut_user ut_name
#define ut_xtime ut_tv.tv_sec
#endif /* _UTMPX_COMPAT */

struct utmpx {
	char ut_user[_UTX_USERSIZE];	/* login name */
	char ut_id[_UTX_IDSIZE];	    /* id */
	char ut_line[_UTX_LINESIZE];	/* tty name */
	pid_t ut_pid;			        /* process id creating the entry */
	short ut_type;			        /* type of this entry */
	struct timeval ut_tv;		    /* time entry was created */
	char ut_host[_UTX_HOSTSIZE];	/* host name */
	__uint32_t ut_pad[16];		    /* reserved for future use */
};

#endif /* !_UTMPX_H_ */