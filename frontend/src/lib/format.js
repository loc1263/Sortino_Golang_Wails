// Funciones de formato compartidas por los componentes.
// Viven aparte para poder reutilizarlas y probarlas sin depender de la interfaz.

/** Convierte una fracción en porcentaje: 0.2331 -> "23.3%". */
export const pct = (value) => (value * 100).toFixed(1) + '%'

/** Número con 2 decimales: 7.7123 -> "7.71". */
export const num = (value) => value.toFixed(2)
