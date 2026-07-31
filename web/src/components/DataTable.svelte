<script>
  /** @type {string[]} */
  export let headers = []
  /**
   * Optional rows as cell arrays. When empty, use the default slot for custom rows.
   * @type {Array<Array<string | number | null | undefined>>}
   */
  export let rows = []
</script>

<div class="overflow-x-auto rounded-xl border border-[#bbf7d0] bg-white shadow-sm" data-testid="data-table">
  <table class="min-w-full divide-y divide-green-100 text-left text-sm">
    <thead class="bg-green-50/80">
      <tr>
        {#each headers as header}
          <th scope="col" class="px-4 py-3 font-semibold text-green-900 whitespace-nowrap">
            {header}
          </th>
        {/each}
      </tr>
    </thead>
    <tbody class="divide-y divide-green-50">
      {#if rows.length > 0}
        {#each rows as row}
          <tr class="hover:bg-green-50/40">
            {#each row as cell}
              <td class="px-4 py-3 text-green-950/90 whitespace-nowrap">{cell ?? '—'}</td>
            {/each}
          </tr>
        {/each}
      {:else}
        <slot />
      {/if}
    </tbody>
  </table>
</div>
