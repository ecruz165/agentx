import Table from 'cli-table3';

export function printTable(
  head: string[],
  rows: string[][],
): void {
  const table = new Table({ head, style: { head: ['cyan'] } });
  for (const row of rows) {
    table.push(row);
  }
  console.log(table.toString());
}
