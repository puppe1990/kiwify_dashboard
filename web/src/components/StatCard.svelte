<script>
  export let title = ''
  export let value = ''
  export let hint = ''
</script>

<div
  class="rounded-xl border border-[#bbf7d0] bg-white p-5 shadow-sm"
  data-testid="stat-card"
>
  <p class="text-sm font-medium text-green-800/80">{title}</p>
  <p class="mt-1 text-2xl font-semibold tracking-tight text-green-950">{value}</p>
  {#if hint}
    <p class="mt-1 text-xs text-green-700/70">{hint}</p>
  {/if}
</div>
