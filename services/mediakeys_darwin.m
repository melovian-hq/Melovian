#import <MediaPlayer/MediaPlayer.h>
#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#include "mediakeys_darwin.h"

extern void goMediaKeyPlay(void);
extern void goMediaKeyPause(void);
extern void goMediaKeyToggle(void);
extern void goMediaKeyNext(void);
extern void goMediaKeyPrev(void);
extern void goMediaKeySeekTo(double positionSec);

static BOOL initialized = NO;

void MediaKeysInit(void) {
    if (initialized) return;
    initialized = YES;

    MPRemoteCommandCenter *center = [MPRemoteCommandCenter sharedCommandCenter];

    [center.playCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        goMediaKeyPlay();
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [center.pauseCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        goMediaKeyPause();
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [center.togglePlayPauseCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        goMediaKeyToggle();
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [center.nextTrackCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        goMediaKeyNext();
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [center.previousTrackCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        goMediaKeyPrev();
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [center.changePlaybackPositionCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        MPChangePlaybackPositionCommandEvent *posEvent = (MPChangePlaybackPositionCommandEvent *)event;
        goMediaKeySeekTo(posEvent.positionTime);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    center.playCommand.enabled = YES;
    center.pauseCommand.enabled = YES;
    center.togglePlayPauseCommand.enabled = YES;
    center.nextTrackCommand.enabled = YES;
    center.previousTrackCommand.enabled = YES;
    center.changePlaybackPositionCommand.enabled = YES;
}

static NSMutableDictionary *MediaKeysBaseInfo(const char *title, const char *artist,
                                              const char *album, const char *artworkURL,
                                              double durationSec, double positionSec,
                                              bool playing) {
    NSMutableDictionary *info = [NSMutableDictionary dictionary];

    if (title) {
        info[MPMediaItemPropertyTitle] = [NSString stringWithUTF8String:title];
    }
    if (artist) {
        info[MPMediaItemPropertyArtist] = [NSString stringWithUTF8String:artist];
    }
    if (album) {
        info[MPMediaItemPropertyAlbumTitle] = [NSString stringWithUTF8String:album];
    }
    if (artworkURL) {
        NSURL *url = [NSURL URLWithString:[NSString stringWithUTF8String:artworkURL]];
        if (url) {
            MPMediaItemArtwork *artwork = [[MPMediaItemArtwork alloc]
                initWithBoundsSize:CGSizeMake(512, 512)
                requestHandler:^NSImage * _Nonnull(CGSize size) {
                    NSData *data = [NSData dataWithContentsOfURL:url];
                    if (!data) return nil;
                    return [[NSImage alloc] initWithData:data];
                }];
            if (artwork) {
                info[MPMediaItemPropertyArtwork] = artwork;
            }
        }
    }
    if (durationSec > 0) {
        info[MPMediaItemPropertyPlaybackDuration] = @(durationSec);
    }
    info[MPNowPlayingInfoPropertyElapsedPlaybackTime] = @(positionSec);
    info[MPNowPlayingInfoPropertyPlaybackRate] = playing ? @(1.0) : @(0.0);

    return info;
}

void MediaKeysUpdateNowPlaying(const char *title, const char *artist,
                               const char *album, const char *artworkURL,
                               double durationSec, double positionSec,
                               bool playing, bool canGoNext, bool canGoPrevious) {
    NSMutableDictionary *info = MediaKeysBaseInfo(title, artist, album, artworkURL,
                                                  durationSec, positionSec, playing);
    MPNowPlayingInfoCenter *center = [MPNowPlayingInfoCenter defaultCenter];
    center.nowPlayingInfo = info;
    center.playbackState = playing ? MPNowPlayingPlaybackStatePlaying
                                   : MPNowPlayingPlaybackStatePaused;

    MPRemoteCommandCenter *commandCenter = [MPRemoteCommandCenter sharedCommandCenter];
    commandCenter.nextTrackCommand.enabled = canGoNext;
    commandCenter.previousTrackCommand.enabled = canGoPrevious;
    commandCenter.changePlaybackPositionCommand.enabled = durationSec > 0;
}

void MediaKeysUpdatePosition(double positionSec, bool playing) {
    NSMutableDictionary *info = [[MPNowPlayingInfoCenter defaultCenter].nowPlayingInfo mutableCopy];
    if (!info) return;
    info[MPNowPlayingInfoPropertyElapsedPlaybackTime] = @(positionSec);
    info[MPNowPlayingInfoPropertyPlaybackRate] = playing ? @(1.0) : @(0.0);
    MPNowPlayingInfoCenter *center = [MPNowPlayingInfoCenter defaultCenter];
    center.nowPlayingInfo = info;
    center.playbackState = playing ? MPNowPlayingPlaybackStatePlaying
                                   : MPNowPlayingPlaybackStatePaused;
}

void MediaKeysClose(void) {
    MPRemoteCommandCenter *center = [MPRemoteCommandCenter sharedCommandCenter];
    [center.playCommand removeTarget:nil];
    [center.pauseCommand removeTarget:nil];
    [center.togglePlayPauseCommand removeTarget:nil];
    [center.nextTrackCommand removeTarget:nil];
    [center.previousTrackCommand removeTarget:nil];
    [center.changePlaybackPositionCommand removeTarget:nil];

    MPNowPlayingInfoCenter *nowPlaying = [MPNowPlayingInfoCenter defaultCenter];
    nowPlaying.nowPlayingInfo = nil;
    nowPlaying.playbackState = MPNowPlayingPlaybackStateStopped;
    initialized = NO;
}
