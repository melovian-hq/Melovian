<script lang="ts">
  import { link, router, withBase } from "./router.svelte";

  interface Props {
    href: string;
    class?: string;
    activeClass?: string;
    matchPrefix?: string;
    children?: import("svelte").Snippet;
    [key: string]: unknown;
  }

  let {
    href,
    class: className = "",
    activeClass = "",
    matchPrefix = "",
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
  use:link={href}
  {...rest}
>
  {@render children?.()}
</a>
