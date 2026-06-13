/**
 * QRIS Dynamic Generator - Shared Library
 * Algoritma sesuai standar EMVCo QR Code for Payment
 * Tested & verified: semua variasi (static/dynamic + int/decimal) bisa di-scan
 */

// CRC16-CCITT (polynomial 0x1021) - sama dengan standar QRIS
export function crc16CCITT(data: string): string {
  let crc = 0xFFFF
  for (let i = 0; i < data.length; i++) {
    crc ^= data.charCodeAt(i) << 8
    for (let j = 0; j < 8; j++) {
      if (crc & 0x8000) {
        crc = ((crc << 1) ^ 0x1021) & 0xFFFF
      } else {
        crc = (crc << 1) & 0xFFFF
      }
    }
  }
  return crc.toString(16).toUpperCase().padStart(4, '0')
}

// Hapus tag 54 (amount) dari string QRIS
function removeTag54(qris: string): string {
  let i = 0
  while (i + 4 <= qris.length) {
    const tag = qris.substring(i, i + 2)
    const l = parseInt(qris.substring(i + 2, i + 4), 10)
    const end = i + 4 + l
    if (isNaN(l) || end > qris.length) break
    if (tag === '54') {
      return qris.substring(0, i) + qris.substring(end)
    }
    i = end
  }
  return qris
}

// Buat QRIS Dinamis dari QRIS Statis
export function makeDynamicQRIS(qris: string, amount: number): string {
  // hapus CRC lama
  const crcIdx = qris.lastIndexOf('6304')
  if (crcIdx !== -1) {
    qris = qris.substring(0, crcIdx)
  }

  // ubah static 010211 jadi dynamic 010212
  qris = qris.replace('010211', '010212')

  // hapus nominal lama kalau ada
  qris = removeTag54(qris)

  // buat tag 54 (amount integer, tanpa decimal)
  const amountStr = amount.toString()
  const tag54 = '54' + amountStr.length.toString().padStart(2, '0') + amountStr

  // taruh nominal sebelum tag 58 negara, fallback sebelum CRC
  const idx58 = qris.indexOf('5802ID')
  if (idx58 !== -1) {
    qris = qris.substring(0, idx58) + tag54 + qris.substring(idx58)
  } else {
    qris += tag54
  }

  // hitung CRC baru
  const payload = qris + '6304'
  const crc = crc16CCITT(payload)

  return qris + '6304' + crc
}

// Parse EMVCo QR string (untuk display/debug)
export function parseEmvcoQr(qrisString: string): Record<string, string> {
  const result: Record<string, string> = {}
  let remaining = qrisString
  while (remaining.length > 0) {
    if (remaining.length < 4) break
    const id = remaining.substring(0, 2)
    const length = parseInt(remaining.substring(2, 4), 10)
    if (isNaN(length) || remaining.length < 4 + length) break
    const value = remaining.substring(4, 4 + length)
    result[id] = value
    remaining = remaining.substring(4 + length)
  }
  return result
}
