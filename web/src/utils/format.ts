// 格式化辅助：金额（分->元）、距离（公里）、容积（毫升->升）、日期。

// 金额分转元字符串，保留两位小数。
export function formatMoney(cents: number | null | undefined): string {
  if (cents === null || cents === undefined) return '-'
  const yuan = cents / 100
  return yuan.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

// 金额分转元，附带“元”单位。
export function formatYuan(cents: number | null | undefined): string {
  if (cents === null || cents === undefined) return '-'
  return `${formatMoney(cents)} 元`
}

// 毫升转升，保留三位小数。
export function formatLiters(milli: number | null | undefined): string {
  if (milli === null || milli === undefined) return '-'
  const liters = milli / 1000
  return liters.toLocaleString('zh-CN', { minimumFractionDigits: 3, maximumFractionDigits: 3 })
}

// 毫升转升字符串（带单位）。
export function formatLitersUnit(milli: number | null | undefined): string {
  if (milli === null || milli === undefined) return '-'
  return `${formatLiters(milli)} L`
}

// 公里数格式化。
export function formatKm(km: number | null | undefined): string {
  if (km === null || km === undefined) return '-'
  return `${km.toLocaleString('zh-CN')} km`
}

// 通用数字格式化。
export function formatNum(n: number | null | undefined): string {
  if (n === null || n === undefined) return '-'
  return n.toLocaleString('zh-CN')
}

// 百分比格式化。
export function formatPct(n: number | null | undefined): string {
  if (n === null || n === undefined) return '-'
  return `${n.toFixed(2)}%`
}

// 日期时间格式化为本地可读字符串。
export function formatDateTime(s: string | null | undefined): string {
  if (!s) return '-'
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  const pad = (x: number) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// 仅日期。
export function formatDate(s: string | null | undefined): string {
  if (!s) return '-'
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  const pad = (x: number) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

// 将本地日期时间（input type=datetime-local）转为 RFC3339 UTC。
export function toRFC3339(local: string): string {
  if (!local) return ''
  const d = new Date(local)
  if (Number.isNaN(d.getTime())) return local
  return d.toISOString()
}

// 将 RFC3339 转为 datetime-local 输入值（本地时区）。
export function fromRFC3339ToLocal(s: string | undefined): string {
  if (!s) return ''
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (x: number) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}
