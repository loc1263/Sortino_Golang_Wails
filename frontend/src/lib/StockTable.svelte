<!--
  StockTable: muestra el ranking en una tabla.

  Es un componente "tonto" (presentacional): no sabe de dónde vienen los
  datos ni llama al backend. Solo recibe `stocks` y los dibuja. Separar así
  la presentación de la lógica hace que sea fácil de reutilizar y de cambiar.
-->
<script>
  import { pct, num } from './format.js'

  // Svelte 5: las propiedades que recibe el componente se declaran con $props().
  /** @type {{ stocks: Array<Record<string, any>> }} */
  let { stocks } = $props()
</script>

<table>
  <thead>
    <tr>
      <th>#</th>
      <th>Símbolo</th>
      <th class="left">Empresa</th>
      <th class="left">Sector</th>
      <th>Precio</th>
      <th>Mom 12-1</th>
      <th>Vol 20d</th>
      <th>Sortino 3M</th>
      <th>Ret 3M</th>
      <th>Caída máx 3M</th>
      <th>Ret / Caída</th>
      <th>Score</th>
    </tr>
  </thead>
  <tbody>
    <!-- La "key" (s.symbol) ayuda a Svelte a reutilizar filas al cambiar la lista. -->
    {#each stocks as s, i (s.symbol)}
      <tr>
        <td>{i + 1}</td>
        <td><strong>{s.symbol}</strong></td>
        <td class="left">{s.name}</td>
        <td class="left">{s.sector}</td>
        <td>{num(s.last)}</td>
        <!-- class={{ neg: condición }} agrega la clase "neg" solo si se cumple -->
        <td class={{ neg: s.momentum < 0 }}>{pct(s.momentum)}</td>
        <td>{pct(s.vol20)}</td>
        <td>{num(s.sortino)}</td>
        <td class={{ neg: s.return3m < 0 }}>{pct(s.return3m)}</td>
        <td>{pct(s.maxdd3m)}</td>
        <td>{num(s.calmar3m)}</td>
        <td><strong>{s.score.toFixed(1)}</strong></td>
      </tr>
    {/each}
  </tbody>
</table>

<!-- Los estilos de un componente Svelte solo afectan a su propio HTML (están "encapsulados"). -->
<style>
  table {
    margin: 0 auto;
    border-collapse: collapse;
    font-size: 0.9rem;
  }

  th,
  td {
    padding: 6px 12px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.15);
    text-align: right;
  }

  th.left,
  td.left {
    text-align: left;
  }

  .neg {
    color: #ff8080;
  }
</style>
