#import <AppKit/AppKit.h>
#import <dispatch/dispatch.h>
#import "appicon_darwin.h"

int BDSetAppIcon(const char *data, int length) {
    if (data == NULL || length <= 0) {
        return 0;
    }

    NSData *raw = [NSData dataWithBytes:data length:(NSUInteger)length];
    NSImage *image = [[NSImage alloc] initWithData:raw];
    if (image == nil) {
        return 0;
    }

    // The Dock icon belongs to the app object, which must be touched from the
    // main thread; the binding can be called from any goroutine. The block keeps
    // a reference to the image, so its release balances the alloc above.
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApplication sharedApplication];
        [NSApp setApplicationIconImage:image];
        [image release];
    });
    return 1;
}
