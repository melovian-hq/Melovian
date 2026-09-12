<script lang="ts">
  // SPDX-FileCopyrightText: 2026 Quad4 Software
  // SPDX-License-Identifier: Apache-2.0

  import { untrack } from "svelte";
  import {
    createAsyncPage,
    type AsyncPageOptions,
  } from "$lib/ui/async-page.svelte";

  let { options }: { options: AsyncPageOptions<string> } = $props();

  const page = createAsyncPage<string>(untrack(() => options));

  export function getPage() {
    return page;
  }
</script>
