<script lang="ts">
  import { link, router, withBase } from "./router.svelte";
  import {
    contextMenuAttachment,
    type ContextMenuPosition,
  } from "$lib/components/ui/context-menu";

  interface Props {
    href: string;
    class?: string;
    activeClass?: string;
    matchPrefix?: string;
    onmenu?: (pos: ContextMenuPosition) => void;
    children?: import("svelte").Snippet;
    [key: string]: unknown;
  }

  let {
    href,
    class: className = "",
    activeClass = "",
    matchPrefix = "",
    onmenu,
    children,
    ...rest
  }: Props = $props();

  const resolvedHref = $derived(withBase(href));

  const active = $derived(
    matchPrefix
      ? router.pathname === matchPrefix ||
          router.pathname.startsWith(`${matchPrefix}/`)
      : router.pathname === href ||
          (href !== "/" && router.pathname.startsWith(href + "/")),
  );
</script>

<a
  href={resolvedHref}
  class="{className}{active && activeClass ? ` ${activeClass}` : ''}"
  aria-current={active ? "page" : undefined}
  use:link={href}
  {@attach contextMenuAttachment(onmenu)}
  {...rest}
>
  {@render children?.()}
</a>
