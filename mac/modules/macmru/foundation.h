#import <Foundation/Foundation.h>
_Pragma("GCC diagnostic ignored \"-Wincompatible-pointer-types\"")

typedef struct _NSURLdata {
    NSString *scheme;
    NSString *user;
    NSString *password;
    NSString *host;
    NSNumber *port;
    NSString *path;
    NSString *query;
    NSString *fragment;
} NSURLdata;

const NSArray* FinderFXRecentFolders();
const char* NSStringToCString(NSString*);
int NSNumberToGoInt(NSNumber*);
const NSURLdata* NSURLData(NSURL*);
inline unsigned long NSArrayLen(NSArray* arr) {
    if (arr == NULL) { return 0; }
    return arr.count;
}
inline const void* NSArrayItem(NSArray* arr, unsigned long i) {
    if (arr == NULL) { return NULL; }
    return [arr objectAtIndex:i];
}