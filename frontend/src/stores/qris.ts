import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface QrisProduct {
  id: string
  name: string
  price: number
  category?: string
}

export interface QrisTransaction {
  id: string
  productId?: string
  productName?: string
  amount: number
  qrDataUrl?: string
  status: 'pending' | 'paid' | 'cancelled'
  createdAt: number
  paidAt?: number
  notes?: string
}

const STORAGE_KEYS = {
  QRIS_STRING: 'wakupi_qris_string',
  PRODUCTS: 'wakupi_qris_products',
  TRANSACTIONS: 'wakupi_qris_transactions',
}

export const useQrisStore = defineStore('qris', () => {
  const qrisString = ref<string>('')
  const products = ref<QrisProduct[]>([])
  const transactions = ref<QrisTransaction[]>([])

  // Load from localStorage on init
  function loadFromStorage() {
    try {
      const storedQris = localStorage.getItem(STORAGE_KEYS.QRIS_STRING)
      const storedProducts = localStorage.getItem(STORAGE_KEYS.PRODUCTS)
      const storedTransactions = localStorage.getItem(STORAGE_KEYS.TRANSACTIONS)

      if (storedQris) qrisString.value = storedQris
      if (storedProducts) products.value = JSON.parse(storedProducts)
      if (storedTransactions) transactions.value = JSON.parse(storedTransactions)
    } catch (e) {
      console.error('Failed to load QRIS data from storage:', e)
    }
  }

  function saveToStorage() {
    try {
      localStorage.setItem(STORAGE_KEYS.QRIS_STRING, qrisString.value)
      localStorage.setItem(STORAGE_KEYS.PRODUCTS, JSON.stringify(products.value))
      localStorage.setItem(STORAGE_KEYS.TRANSACTIONS, JSON.stringify(transactions.value))
    } catch (e) {
      console.error('Failed to save QRIS data to storage:', e)
    }
  }

  function setQrisString(str: string) {
    qrisString.value = str
    saveToStorage()
  }

  function addProduct(product: Omit<QrisProduct, 'id'>) {
    const newProduct: QrisProduct = {
      id: `prod-${Date.now()}`,
      ...product,
    }
    products.value.push(newProduct)
    saveToStorage()
    return newProduct
  }

  function updateProduct(id: string, updates: Partial<QrisProduct>) {
    const index = products.value.findIndex((p) => p.id === id)
    if (index >= 0) {
      products.value[index] = { ...products.value[index], ...updates }
      saveToStorage()
    }
  }

  function deleteProduct(id: string) {
    products.value = products.value.filter((p) => p.id !== id)
    saveToStorage()
  }

  function addTransaction(transaction: Omit<QrisTransaction, 'id' | 'createdAt' | 'status'>) {
    const newTransaction: QrisTransaction = {
      id: `txn-${Date.now()}`,
      ...transaction,
      status: 'pending',
      createdAt: Date.now(),
    }
    transactions.value.unshift(newTransaction)
    saveToStorage()
    return newTransaction
  }

  function updateTransactionStatus(id: string, status: QrisTransaction['status']) {
    const transaction = transactions.value.find((t) => t.id === id)
    if (transaction) {
      transaction.status = status
      if (status === 'paid') transaction.paidAt = Date.now()
      saveToStorage()
    }
  }

  function deleteTransaction(id: string) {
    transactions.value = transactions.value.filter((t) => t.id !== id)
    saveToStorage()
  }

  function clearAllData() {
    qrisString.value = ''
    products.value = []
    transactions.value = []
    localStorage.removeItem(STORAGE_KEYS.QRIS_STRING)
    localStorage.removeItem(STORAGE_KEYS.PRODUCTS)
    localStorage.removeItem(STORAGE_KEYS.TRANSACTIONS)
  }

  // Stats
  const todayTransactions = computed(() => {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    const todayStart = today.getTime()
    return transactions.value.filter((t) => t.createdAt >= todayStart)
  })

  const totalToday = computed(() =>
    todayTransactions.value.filter((t) => t.status === 'paid').reduce((sum, t) => sum + t.amount, 0)
  )

  const totalPending = computed(() =>
    transactions.value.filter((t) => t.status === 'pending').reduce((sum, t) => sum + t.amount, 0)
  )

  // Initialize
  loadFromStorage()

  return {
    qrisString,
    products,
    transactions,
    setQrisString,
    addProduct,
    updateProduct,
    deleteProduct,
    addTransaction,
    updateTransactionStatus,
    deleteTransaction,
    clearAllData,
    todayTransactions,
    totalToday,
    totalPending,
  }
})
