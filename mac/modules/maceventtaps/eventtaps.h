#include <ApplicationServices/ApplicationServices.h>
#include <CoreGraphics/CoreGraphics.h>
#include <Foundation/Foundation.h>
_Pragma("GCC diagnostic ignored \"-Wincompatible-pointer-types\"")

const NSArray* GetEventTapList();
NSString* pathFromPid(pid_t);
const NSString* NSDictionaryValueForKey(NSDictionary*, NSString*);
NSString* CStringToNSString(char*);
inline unsigned long NSArrayLen(NSArray* arr) {
    if (arr == NULL) { return 0; }
    return arr.count;
}
inline const void* NSArrayItem(NSArray* arr, unsigned long i) {
    if (arr == NULL) { return NULL; }
    return [arr objectAtIndex:i];
}
inline const char* NSStringToCString(NSString* s) {
    if (s == NULL) { return NULL; }

    const char *cstr = [s UTF8String];
    return cstr;
}
inline int NSNumberToGoInt(NSNumber* i) {
    if (i == NULL) { return 0; }
    return i.intValue;
}