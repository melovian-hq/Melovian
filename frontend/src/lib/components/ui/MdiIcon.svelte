<script lang="ts">
  import Icon from "@iconify/svelte";

  const ICONS: Record<string, string> = {
    check: "mdi:check",
    cloudCheck: "mdi:cloud-check-outline",
    cloudOff: "mdi:cloud-off-outline",
    chevronDown: "mdi:chevron-down",
    chevronLeft: "mdi:chevron-left",
    chevronRight: "mdi:chevron-right",
    chevronUp: "mdi:chevron-up",
    dice: "mdi:dice-5",
    disc: "mdi:disc",
    download: "mdi:download-outline",
    downloadDone: "mdi:check-circle",
    export: "mdi:export-variant",
    upload: "mdi:upload",
    fullscreen: "mdi:fullscreen",
    fullscreenExit: "mdi:fullscreen-exit",
    folderMusic: "mdi:folder-music-outline",
    folderOpen: "mdi:folder-open-outline",
    folderPlus: "mdi:folder-plus-outline",
    gripVertical: "mdi:drag-vertical",
    resizeBottomRight: "mdi:resize-bottom-right",
    resizeBottomLeft: "mdi:resize-bottom-left",
    headphones: "mdi:headphones",
    heartPulse: "mdi:heart-pulse",
    listMusic: "mdi:playlist-music",
    viewList: "mdi:view-list",
    viewGrid: "mdi:view-grid",
    viewCard: "mdi:cards-playing-outline",
    menu: "mdi:menu",
    dotsHorizontal: "mdi:dots-horizontal",
    mic2: "mdi:microphone-variant",
    lyrics: "mdi:text-box-outline",
    moon: "mdi:moon-waning-crescent",
    monitor: "mdi:monitor",
    music: "mdi:music",
    video: "mdi:video-outline",
    panelLeftClose: "mdi:page-layout-sidebar-left",
    panelLeftOpen: "mdi:page-layout-body",
    pause: "mdi:pause",
    pencil: "mdi:pencil-outline",
    play: "mdi:play",
    playNext: "mdi:playlist-play",
    plus: "mdi:plus",
    queueAdd: "mdi:playlist-plus",
    radio: "mdi:radio",
    recordCircle: "mdi:record-circle",
    refresh: "mdi:refresh",
    scrollFollow: "mdi:arrow-down-bold-circle-outline",
    scrollFollowActive: "mdi:arrow-down-bold-circle",
    repeat: "mdi:repeat",
    repeatOnce: "mdi:repeat-once",
    search: "mdi:magnify",
    server: "mdi:server",
    settings: "mdi:cog-outline",
    shuffle: "mdi:shuffle",
    signal: "mdi:access-point",
    skipBack: "mdi:skip-previous",
    skipForward: "mdi:skip-next",
    slidersHorizontal: "mdi:tune-variant",
    squareCheck: "mdi:checkbox-marked-outline",
    star: "mdi:star",
    sun: "mdi:white-balance-sunny",
    tag: "mdi:tag-multiple-outline",
    trash2: "mdi:delete-outline",
    trendingUp: "mdi:trending-up",
    chartBar: "mdi:chart-bar",
    chartRadar: "mdi:chart-donut-variant",
    circleSmall: "mdi:circle-small",
    history: "mdi:history",
    keyboard: "mdi:keyboard-outline",
    lightbulb: "mdi:lightbulb-outline",
    eye: "mdi:eye-outline",
    eyeOff: "mdi:eye-off-outline",
    clockOutline: "mdi:clock-outline",
    album: "mdi:album",
    accountMusic: "mdi:account-music",
    alertCircle: "mdi:alert-circle-outline",
    alert: "mdi:alert-outline",
    info: "mdi:information-outline",
    bell: "mdi:bell-outline",
    bellBadge: "mdi:bell-badge-outline",
    autoFix: "mdi:auto-fix",
    puzzle: "mdi:puzzle-outline",
    contentSave: "mdi:content-save-outline",
    share: "mdi:share-variant-outline",
    link: "mdi:link-variant",
    lock: "mdi:lock-outline",
    accountMultiple: "mdi:account-multiple-outline",
    harddisk: "mdi:harddisk",
    volume2: "mdi:volume-high",
    volumeLow: "mdi:volume-low",
    volumeMedium: "mdi:volume-medium",
    volumeOff: "mdi:volume-off",
    rewind: "mdi:rewind",
    fastForward: "mdi:fast-forward",
    x: "mdi:close",
  };

  interface Props {
    name: keyof typeof ICONS | string;
    size?: number;
    class?: string;
    fill?: string;
    "aria-hidden"?: boolean | "true" | "false";
  }

  let {
    name,
    size = 24,
    class: className = "",
    fill,
    "aria-hidden": ariaHidden,
  }: Props = $props();

  const icon = $derived(ICONS[name] ?? name);
</script>

<Icon
  {icon}
  width={size}
  height={size}
  class={className}
  {fill}
  aria-hidden={ariaHidden}
/>
