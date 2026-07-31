<script>
  import AppLayout from '../components/AppLayout.svelte'
  import StatCard from '../components/StatCard.svelte'
  import DataTable from '../components/DataTable.svelte'

  export let totalContacts = 0
  export let env = ''
  export let site = {}
  export let flash = {}
</script>

<svelte:head>
  <title>Dashboard · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6">
    <h1 class="text-2xl font-semibold text-green-950">Dashboard</h1>
    <p class="mt-1 text-sm text-green-800/70">Visão geral das operações Kiwify</p>
  </div>

  <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
    <StatCard title="Vendas (30d)" value="—" hint="Dados em breve" />
    <StatCard title="Receita (30d)" value="—" hint="Dados em breve" />
    <StatCard title="Saldo" value="—" hint="Dados em breve" />
    <StatCard title="Eventos recentes" value="—" hint="Dados em breve" />
  </div>

  <div class="mt-8 grid gap-6 lg:grid-cols-2">
    <section>
      <h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-green-800/80">
        Últimas vendas
      </h2>
      <DataTable
        headers={['Referência', 'Status', 'Valor', 'Data']}
        rows={[]}
      >
        <tr>
          <td colspan="4" class="px-4 py-8 text-center text-sm text-green-700/60">
            Nenhuma venda carregada ainda.
          </td>
        </tr>
      </DataTable>
    </section>

    <section>
      <h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-green-800/80">
        Eventos recentes
      </h2>
      <div class="rounded-xl border border-[#bbf7d0] bg-white p-5 shadow-sm">
        <p class="text-sm text-green-700/60">Feed de webhooks em breve.</p>
        {#if env}
          <p class="mt-3 text-xs text-green-600/50">Env: {env} · Contatos: {totalContacts}</p>
        {/if}
      </div>
    </section>
  </div>
</AppLayout>
