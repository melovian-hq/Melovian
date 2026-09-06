#ifndef MEDIAKEYS_DARWIN_H
#define MEDIAKEYS_DARWIN_H

#include <stdbool.h>

void MediaKeysInit(void);
void MediaKeysUpdateNowPlaying(const char *title, const char *artist,
                               const char *album, const char *artworkURL,
                               double durationSec, double positionSec,
                               bool playing, bool canGoNext, bool canGoPrevious);
void MediaKeysUpdatePosition(double positionSec, bool playing);
void MediaKeysClose(void);

#endif
