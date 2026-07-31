<script>
  /** Inertia flash props: success | error | notice | warning */
  export let flash = {}
</script>

{#if flash?.success}
  <div
    class="mb-4 rounded-lg border border-green-200 bg-green-50 px-4 py-2.5 text-sm text-green-900"
    data-testid="flash-success"
    role="status"
  >
    {flash.success}
  </div>
{/if}
{#if flash?.notice}
  <div
    class="mb-4 rounded-lg border border-green-200 bg-green-50 px-4 py-2.5 text-sm text-green-900"
    data-testid="flash-notice"
    role="status"
  >
    {flash.notice}
  </div>
{/if}
{#if flash?.error}
  <div
    class="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-2.5 text-sm text-red-800"
    data-testid="flash-error"
    role="alert"
  >
    {flash.error}
  </div>
{/if}
{#if flash?.warning}
  <div
    class="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-2.5 text-sm text-amber-900"
    data-testid="flash-warning"
    role="status"
  >
    {flash.warning}
  </div>
{/if}
