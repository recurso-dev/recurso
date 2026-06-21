import React, { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'

const API_BASE = import.meta.env.VITE_API_BASE_URL?.replace('/v1', '') || 'http://localhost:8080'

export default function Checkout() {
  const { id } = useParams()
  const [invoice, setInvoice] = useState(null)
  const [loading, setLoading] = useState(true)
  const [paying, setPaying] = useState(false)
  const [error, setError] = useState(null)
  const [success, setSuccess] = useState(false)

  useEffect(() => {
    fetch(`${API_BASE}/checkout/${id}`)
      .then(res => {
        if (!res.ok) throw new Error('Invoice not found')
        return res.json()
      })
      .then(data => {
        setInvoice(data.data)
        if (data.data.status === 'paid') {
          setSuccess(true)
        }
      })
      .catch(err => setError(err.message))
      .finally(() => setLoading(false))
  }, [id])

  const handlePay = async () => {
    setPaying(true)
    setError(null)
    try {
      const res = await fetch(`${API_BASE}/checkout/${id}/pay`, { method: 'POST' })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || 'Payment failed')
      }
      const orderData = await res.json()

      // Mark as success via the success endpoint
      const successRes = await fetch(`${API_BASE}/checkout/${id}/success`)
      if (successRes.ok) {
        setSuccess(true)
      }
    } catch (err) {
      setError(err.message)
    } finally {
      setPaying(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-gray-900 border-t-transparent" />
      </div>
    )
  }

  if (error && !invoice) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="bg-white rounded-xl shadow-lg p-8 max-w-md w-full text-center">
          <div className="text-red-500 text-4xl mb-4">!</div>
          <h1 className="text-xl font-bold text-gray-900 mb-2">Invoice Not Found</h1>
          <p className="text-gray-500">{error}</p>
        </div>
      </div>
    )
  }

  if (success) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="bg-white rounded-xl shadow-lg p-8 max-w-md w-full text-center">
          <div className="text-green-500 text-5xl mb-4">&#10003;</div>
          <h1 className="text-2xl font-bold text-gray-900 mb-2">Payment Successful</h1>
          <p className="text-gray-500 mb-4">
            Invoice {invoice?.invoice_number} has been paid.
          </p>
          <p className="text-sm text-gray-400">You can close this page.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-xl shadow-lg p-8 max-w-md w-full">
        <div className="text-center mb-6">
          <div className="inline-block w-12 h-12 bg-gray-900 rounded-xl text-white text-2xl font-bold leading-[48px] mb-4">
            R
          </div>
          <h1 className="text-2xl font-bold text-gray-900">Checkout</h1>
        </div>

        <div className="bg-gray-50 rounded-lg p-4 mb-6 space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-500 text-sm">Invoice</span>
            <span className="font-semibold text-gray-900 text-sm">{invoice.invoice_number}</span>
          </div>
          {invoice.subtotal !== invoice.total && (
            <>
              <div className="flex justify-between">
                <span className="text-gray-500 text-sm">Subtotal</span>
                <span className="text-gray-900 text-sm">
                  {invoice.currency} {(invoice.subtotal / 100).toFixed(2)}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500 text-sm">Tax</span>
                <span className="text-gray-900 text-sm">
                  {invoice.currency} {(invoice.tax_amount / 100).toFixed(2)}
                </span>
              </div>
              <div className="border-t pt-2" />
            </>
          )}
          <div className="flex justify-between">
            <span className="text-gray-500 text-sm">Total</span>
            <span className="font-bold text-gray-900 text-lg">
              {invoice.currency} {invoice.display_amount}
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-500 text-sm">Due Date</span>
            <span className="text-gray-900 text-sm">{invoice.due_date}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-500 text-sm">Status</span>
            <span className="inline-block px-2 py-0.5 bg-yellow-100 text-yellow-800 text-xs font-semibold rounded-full uppercase">
              {invoice.status}
            </span>
          </div>
        </div>

        {error && (
          <div className="bg-red-50 text-red-700 text-sm rounded-lg p-3 mb-4">
            {error}
          </div>
        )}

        <button
          onClick={handlePay}
          disabled={paying}
          className="w-full bg-gray-900 text-white py-3 rounded-lg font-semibold hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {paying ? 'Processing...' : `Pay ${invoice.currency} ${invoice.display_amount}`}
        </button>

        <p className="text-center text-xs text-gray-400 mt-4">
          Powered by Recurso
        </p>
      </div>
    </div>
  )
}
