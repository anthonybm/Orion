#import "foundation.h"

const NSArray*
FinderFXRecentFolders() {
    CFPreferencesAppSynchronize(CFSTR("com.apple.finder"));
    NSMutableArray * urlArrays = [[NSMutableArray alloc] init];
    NSArray* folderList = (__bridge_transfer NSArray*) CFPreferencesCopyAppValue(CFSTR("FXRecentFolders"), CFSTR("com.apple.finder"));
    for (NSDictionary* currentFolder in folderList) {
        // NSLog(@"Name: %@", [currentFolder objectForKey:@"name"]);

        NSURL* folderURL = [NSURL URLByResolvingBookmarkData:[currentFolder objectForKey:@"file-bookmark"]
        options:NSURLBookmarkResolutionWithoutUI | NSURLBookmarkResolutionWithoutMounting
        relativeToURL: nil
        bookmarkDataIsStale: nil
        error: nil];

        // NSLog(@"Path: %@", [folderURL path]);

        NSURL* url = folderURL;
        if (url != nil) {   // handle nil case
            [urlArrays addObject: url];
        }
    }
    return urlArrays;
}

const char* NSStringToCString(NSString* s) {
    if (s == NULL) { return NULL; }

    const char *cstr = [s UTF8String];
    return cstr;
}
int NSNumberToGoInt(NSNumber* i) {
    if (i == NULL) { return 0; }
    return i.intValue;
}
const NSURLdata* NSURLData(NSURL* url) {
    NSURLdata *urldata = malloc(sizeof(NSURLdata));
    urldata->scheme = url.scheme;
    urldata->user = url.user;
    urldata->password = url.password;
    urldata->host = url.host;
    urldata->port = url.port;
    urldata->path = url.path;
    urldata->query = url.query;
    urldata->fragment = url.fragment;
    return urldata;
}