<script lang="ts">
  // SPDX-FileCopyrightText: 2026 Quad4 Software
  // SPDX-License-Identifier: Apache-2.0

  import { untrack } from "svelte";
  import {
    createPagedList,
    type PagedListOptions,
  } from "$lib/ui/async-page.svelte";

  let { options }: { options: PagedListOptions<string> } = $props();

  const list = createPagedList<string>(untrack(() => options));

  export function getList() {
    return list;
  }
</script>
