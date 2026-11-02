<!--
  App: componente raíz. Maneja el estado de la pantalla y llama al backend en Go.

  Flujo:
    1. El usuario pulsa "Bajar precios"  -> se llama a Backend.TopStocks() (Go)
    2. Go emite eventos "progress" mientras descarga -> se actualiza el contador
    3. Go devuelve el ranking              -> se muestra en <StockTable>
    4. El usuario pulsa "Exportar CSV"   -> Go abre el diálogo y guarda el archivo
-->
<script>
  import { onMount } from 'svelte'
  import { Events } from '@wailsio/runtime'
  // Wails genera este módulo a partir de los métodos públicos de StockService
  // (ver stocks_service.go). Llamar a Backend.X() ejecuta la función Go X().
  import { StockService as Backend } from '../bindings/Sortino_Golang_Wails'
  import StockTable from './lib/StockTable.svelte'

  // --- Estado ---------------------------------------------------------
  // $state crea variables reactivas: al cambiar su valor, Svelte actualiza
  // automáticamente las partes de la pantalla que las usan.
  let loading = $state(false)
  let progress = $state({ done: 0, total: 0 })
  let report = $state(null) // resultado de Go; null hasta la primera descarga
  let error = $state('')
  let savedPath = $state('')

  // --- Eventos del backend -------------------------------------------
  // onMount se ejecuta una vez cuando el componente aparece en pantalla.
  // Events.On devuelve una función para desuscribirse; devolverla desde
  // onMount hace que Svelte la llame al destruir el componente.
  onMount(() => Events.On('progress', (event) => (progress = event.data)))

  // --- Acciones -------------------------------------------------------
  async function loadStocks() {
    loading = true
    error = ''
    savedPath = ''
    progress = { done: 0, total: 0 }
    try {
      // Las funciones de Go devuelven promesas: `await` espera el resultado.
      // Si Go devuelve un error, la promesa se rechaza y cae en el catch.
      report = await Backend.TopStocks()
    } catch (err) {
      error = String(err)
    } finally {
      loading = false // se ejecuta siempre, haya éxito o error
    }
  }

  async function exportCsv() {
    error = ''
    try {
      // Devuelve la ruta guardada, o "" si el usuario canceló el diálogo.
      savedPath = await Backend.ExportCSV()
    } catch (err) {
      error = String(err)
    }
  }
</script>

<main>
  <button class="btn" onclick={loadStocks} disabled={loading}>
    {loading ? 'Descargando…' : 'Bajar precios S&P 500 (último año)'}
  </button>

  {#if loading}
    <p class="status">
      Consultando Yahoo Finance… {progress.done} / {progress.total || '?'}
    </p>
  {/if}

  {#if error}
    <p class="status error">{error}</p>
  {/if}

  <!-- Se muestra solo cuando ya hay resultado y no se está descargando de nuevo. -->
  {#if report && !loading}
    <p class="status">
      Top 10 por score: promedio del percentil de momentum 12-1 (alto),
      volatilidad 20d (baja) y Sortino 3M (alto).
      {report.total - report.failed} de {report.total} acciones procesadas.
    </p>

    <StockTable stocks={report.top} />

    <p class="status">
      <button class="btn" onclick={exportCsv}>Exportar CSV</button>
    </p>
    {#if savedPath}
      <p class="status">Guardado en {savedPath}</p>
    {/if}
  {/if}
</main>

<style>
  main {
    padding: 1.5rem;
  }

  .btn {
    height: 30px;
    border-radius: 3px;
    border: none;
    padding: 0 16px;
    cursor: pointer;
  }

  .btn:hover:not(:disabled) {
    background-image: linear-gradient(to top, #cfd9df 0%, #e2ebf0 100%);
    color: #333333;
  }

  .btn:disabled {
    cursor: wait;
    opacity: 0.6;
  }

  .status {
    margin: 1rem 0;
    font-size: 0.9rem;
  }

  .error {
    color: #ff8080;
  }
</style>
