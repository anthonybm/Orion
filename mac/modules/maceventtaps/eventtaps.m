#import "eventtaps.h"
#import <notify.h>
#import <libproc.h>

// path from the pid
NSString* pathFromPid(pid_t pid)
{
    // buffer the process path
    char pathBuffer[PROC_PIDPATHINFO_MAXSIZE] = {0};
    
    proc_pidpath(pid, pathBuffer, sizeof(pathBuffer));
    
    return [NSString stringWithUTF8String:pathBuffer];
}

const NSArray* GetEventTapList(){
    NSMutableArray* tapsResults = [[NSMutableArray alloc] init];

    uint32_t tapCount = 0;

    CGEventTapInformation* taps = NULL;

    CGEventTapInformation tap = {0};

    NSString* tappingProcess = nil;
    NSString* processBeingTapped = nil;

    // get all taps
    if (kCGErrorSuccess != CGGetEventTapList(0, NULL, &tapCount))
    {
        // bail
        goto bail;
    }

    // debug message
    // NSLog(@"found %d taps", tapCount);

    // allocate
    taps = malloc(sizeof(CGEventTapInformation) * tapCount);
    if (NULL == taps)
    {
        // bail
        goto bail;
    }
    
    // get all taps
    if (kCGErrorSuccess != CGGetEventTapList(tapCount, taps, &tapCount))
    {
        // bail
        goto bail;
    }
    
    // interate through and process all taps
    for (int i = 0; i < tapCount; i++)
    {
        tap = taps[i];
        
        if (true != tap.enabled)
        {
            // skip
            continue;
        }
        
        // // ignore non-keypresses
        // if ( (keyboardTap & tap.eventsOfInterest) != keyboardTap)
        // {
        //     // skip
        //     continue;
        // }
        
        // get path to process
        tappingProcess = pathFromPid(tap.tappingProcess);
        processBeingTapped = pathFromPid(tap.processBeingTapped);
        
        
        
        // add
        //  /*
        //  * Structure used to report information on event taps
        //  */
        // typedef struct CGEventTapInformation
        // {
        //     uint32_t		eventTapID;
        //     CGEventTapLocation	tapPoint;		/* HID, session, annotated session */
        //     CGEventTapOptions	options;		/* Listener, Filter */
        //     CGEventMask		eventsOfInterest;	/* Mask of events being tapped */
        //     pid_t		tappingProcess;		/* Process that is tapping events */
        //     pid_t		processBeingTapped;	/* Zero if not a per-process tap */
        //     bool		enabled;		/* True if tap is enabled */
        //     float		minUsecLatency;		/* Minimum latency in microseconds */
        //     float		avgUsecLatency;		/* Average latency in microseconds */
        //     float		maxUsecLatency;		/* Maximum latency in microseconds */
        // } CGEventTapInformation;
        tapsResults[i] = @{
            @"tappingProcess": tappingProcess, 
            // @"processBeingTapped": @(tap.processBeingTapped).stringValue,
            @"processBeingTapped": processBeingTapped,
            @"enabled": @(tap.enabled).stringValue,
            @"tapPoint": @(tap.tapPoint).stringValue,
            @"eventTapID": @(tap.eventTapID).stringValue,
            @"options": @(tap.options).stringValue,
            @"eventsOfInterest": @(tap.eventsOfInterest).stringValue,
            @"minUsecLatency": @(tap.minUsecLatency).stringValue,
            @"avgUsecLatency": @(tap.avgUsecLatency).stringValue,
            @"maxUsecLatency": @(tap.maxUsecLatency).stringValue
        };
    }
    
bail:
    // for (int i = 0; i < tapCount; i++) {
    //     NSLog(@"%@", tapsResults[i]);
    // }
    return tapsResults;
    
}

// TODO use NSObjects
const NSString* NSDictionaryValueForKey(NSDictionary* tapResult, NSString *s) {
    if (s == NULL) {return NULL;}
    return [tapResult objectForKey:s];
}

NSString* CStringToNSString(char* str) {
    if (str == NULL) {return NULL;}
    NSString* strFromCstr = [[NSString alloc] initWithUTF8String:str];
    return strFromCstr;
}