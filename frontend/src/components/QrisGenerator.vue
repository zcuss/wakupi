<script setup lang="ts">
import { ref, computed } from 'vue'
import jsQR from 'jsqr'
import QRCode from 'qrcode'
import { X, Upload, QrCode, Calculator } from '@lucide/vue'
import { makeDynamicQRIS, parseEmvcoQr } from '../lib/qris'

const emit = defineEmits<{
  close: []
  selectChat: [chatId: string, amount: number]
}>()

const step = ref<'upload' | 'amount' | 'result'>('upload')
const originalQrisString = ref('')
const amount = ref<number>(0)
const generatedQrDataUrl = ref('')
const error = ref('')
const isProcessing = ref(false)

async function handleFileUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]

  if (!file) return

  isProcessing.value = true
  error.value = ''

  try {
    // Read image as data URL
    const reader = new FileReader()
    const imageData = await new Promise<ImageData>((resolve, reject) => {
      reader.onload = () => {
        const img = new Image()
        img.onload = () => {
          // Draw to canvas to get pixel data
          const canvas = document.createElement('canvas')
          canvas.width = img.width
          canvas.height = img.height
          const ctx = canvas.getContext('2d')!
          ctx.drawImage(img, 0, 0)
          const data = ctx.getImageData(0, 0, img.width, img.height)
          resolve(data)
        }
        img.onerror = reject
        img.src = reader.result as string
      }
      reader.onerror = reject
    })

    // Decode QR code
    const qrCode = jsQR(imageData.data, imageData.width, imageData.height)

    if (!qrCode) {
      error.value = 'Tidak dapat membaca QR code dari gambar'
      return
    }

    originalQrisString.value = qrCode.data

    // Validate it's a QRIS string (check for required fields)
    const parts = parseEmvcoQr(qrCode.data)
    if (!parts['00']) {
      error.value = 'QR ini bukan QRIS valid'
      return
    }

    step.value = 'amount'
  } catch (e: any) {
    error.value = 'Gagal memproses gambar: ' + e.message
  } finally {
    isProcessing.value = false
  }
}

async function generateQr() {
  if (!originalQrisString.value || amount.value <= 0) {
    error.value = 'Masukkan nominal yang valid'
    return
  }

  isProcessing.value = true
  error.value = ''

  try {
    // Modify the QRIS string with new amount
    const newQrisString = makeDynamicQRIS(originalQrisString.value, amount.value)

    // Generate QR code as data URL
    generatedQrDataUrl.value = await QRCode.toDataURL(newQrisString, {
      width: 280,
      margin: 2,
      errorCorrectionLevel: 'M'
    })

    step.value = 'result'
  } catch (e: any) {
    error.value = 'Gagal generate QR: ' + e.message
  } finally {
    isProcessing.value = false
  }
}

function reset() {
  step.value = 'upload'
  originalQrisString.value = ''
  amount.value = 0
  generatedQrDataUrl.value = ''
  error.value = ''
}

function formatRupiah(value: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0
  }).format(value)
}

// Quick amount buttons
const quickAmounts = [10000, 25000, 50000, 100000, 150000, 200000]

function setQuickAmount(val: number) {
  amount.value = val
}
</script>

<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
    <div class="bg-white dark:bg-gray-800 rounded-xl w-full max-w-md shadow-2xl">
      <!-- Header -->
      <div class="flex items-center justify-between p-4 border-b dark:border-gray-700">
        <div class="flex items-center gap-2">
          <Calculator class="w-5 h-5 text-wa-green" />
          <h2 class="font-semibold text-lg">Generator QRIS Dinamis</h2>
        </div>
        <button @click="emit('close')" class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded">
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Content -->
      <div class="p-4">
        <!-- Step 1: Upload -->
        <div v-if="step === 'upload'" class="space-y-4">
          <p class="text-sm text-gray-600 dark:text-gray-400">
            Upload QRIS statis dari aplikasi pembayaran Anda
          </p>

          <label class="block">
            <div class="border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-8 text-center cursor-pointer hover:border-wa-green transition-colors">
              <Upload class="w-10 h-10 mx-auto text-gray-400 mb-2" />
              <p class="text-sm text-gray-500">Klik untuk upload QRIS</p>
              <p class="text-xs text-gray-400 mt-1">Format: JPG, PNG</p>
            </div>
            <input type="file" accept="image/*" @change="handleFileUpload" class="hidden" />
          </label>
        </div>

        <!-- Step 2: Input Amount -->
        <div v-else-if="step === 'amount'" class="space-y-4">
          <p class="text-sm text-gray-600 dark:text-gray-400">
            QRIS berhasil dibaca! Masukkan nominal yang diinginkan:
          </p>

          <!-- QR Preview -->
          <div class="flex justify-center p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
            <QrCode class="w-16 h-16 text-gray-400" />
          </div>

          <!-- Amount Input -->
          <div>
            <label class="block text-sm font-medium mb-1">Nominal (Rp)</label>
            <input
              v-model.number="amount"
              type="number"
              min="1000"
              step="100"
              placeholder="Masukkan nominal"
              class="w-full px-4 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 focus:ring-2 focus:ring-wa-green focus:border-transparent"
            />
          </div>

          <!-- Quick Amount Buttons -->
          <div class="flex flex-wrap gap-2">
            <button
              v-for="val in quickAmounts"
              :key="val"
              @click="setQuickAmount(val)"
              class="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 hover:bg-wa-green hover:text-white rounded-full transition-colors"
            >
              {{ formatRupiah(val) }}
            </button>
          </div>

          <button
            @click="generateQr"
            :disabled="isProcessing || amount <= 0"
            class="w-full bg-wa-green hover:bg-wa-green-dark disabled:opacity-50 text-white py-2 rounded-lg font-medium transition-colors"
          >
            {{ isProcessing ? 'Memproses...' : 'Generate QR' }}
          </button>
        </div>

        <!-- Step 3: Result -->
        <div v-else-if="step === 'result'" class="space-y-4">
          <p class="text-sm text-gray-600 dark:text-gray-400">
            QRIS dinamis berhasil dibuat! Tunjukkan ke pelanggan:
          </p>

          <!-- Generated QR -->
          <div class="flex justify-center p-4 bg-white rounded-lg">
            <img :src="generatedQrDataUrl" alt="QRIS Dinamis" class="w-64 h-64" />
          </div>

          <div class="text-center">
            <p class="text-2xl font-bold text-wa-green">{{ formatRupiah(amount) }}</p>
            <p class="text-sm text-gray-500">Silakan scan dalam 24 jam</p>
          </div>

          <button
            @click="reset"
            class="w-full border border-gray-300 dark:border-gray-600 py-2 rounded-lg font-medium hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
          >
            Buat QR Baru
          </button>
        </div>

        <!-- Error Message -->
        <div v-if="error" class="mt-4 p-3 bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded-lg">
          {{ error }}
        </div>
      </div>
    </div>
  </div>
</template>