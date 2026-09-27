#ifndef APPICON_DARWIN_H
#define APPICON_DARWIN_H

#include <stdlib.h>

/* Replaces the running app's Dock icon with a PNG held in memory.
   Returns 1 on success, 0 when the bytes are not a usable image. */
int BDSetAppIcon(const char *data, int length);

#endif
