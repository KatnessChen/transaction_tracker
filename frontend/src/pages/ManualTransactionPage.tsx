import { useState, useCallback, useMemo, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { ROUTES, CURRENCY } from '@/constants'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dropdown, DropdownItem } from '@/components/ui/dropdown'
import DropdownTrigger from '@/components/ui/dropdown-trigger'
import { Label } from '@/components/ui/label'
import { Title } from '@/components/ui/title'
import type { TransactionData, TradeType } from '@/types'
import { TRADE_TYPE } from '@/constants'

interface Currency {
  code: string
  name: string
}

interface Broker {
  id: string
  name: string
}

interface Symbol {
  symbol: string
  name: string
}

export default function ManualTransactionPage() {
  const navigate = useNavigate()
  const [transaction, setTransaction] = useState<TransactionData>({
    id: '',
    transaction_date: '',
    symbol: '',
    trade_type: TRADE_TYPE.BUY as TradeType,
    price: 0,
    quantity: 0,
    amount: 0,
    broker: '',
    currency: CURRENCY.USD,
    exchange: '',
    user_notes: '',
  })

  // Batch state management
  const [batch, setBatch] = useState<TransactionData[]>([])

  // API data states
  const [currencies, setCurrencies] = useState<Currency[]>([])
  const [brokers, setBrokers] = useState<Broker[]>([])
  const [symbols, setSymbols] = useState<Symbol[]>([])
  const [loading, setLoading] = useState({
    currencies: false,
    brokers: false,
    symbols: false,
  })

  // Fetch API data
  useEffect(() => {
    // Load existing batch from localStorage
    const storedBatch = localStorage.getItem('transaction-batch')
    if (storedBatch) {
      setBatch(JSON.parse(storedBatch))
    }

    const fetchCurrencies = async () => {
      setLoading((prev) => ({ ...prev, currencies: true }))
      try {
        // TODO: Replace with actual API endpoint
        const response = await fetch('/api/currencies')
        if (response.ok) {
          const data = await response.json()
          setCurrencies(data)
        } else {
          // Fallback to default currencies if API fails
          setCurrencies([
            { code: CURRENCY.USD, name: 'US Dollar' },
            { code: CURRENCY.CAD, name: 'Canadian Dollar' },
          ])
        }
      } catch (error) {
        console.error('Failed to fetch currencies:', error)
        // Fallback to default currencies
        setCurrencies([
          { code: CURRENCY.USD, name: 'US Dollar' },
          { code: CURRENCY.CAD, name: 'Canadian Dollar' },
        ])
      } finally {
        setLoading((prev) => ({ ...prev, currencies: false }))
      }
    }

    const fetchBrokers = async () => {
      setLoading((prev) => ({ ...prev, brokers: true }))
      try {
        // TODO: Replace with actual API endpoint
        const response = await fetch('/api/brokers')
        if (response.ok) {
          const data = await response.json()
          setBrokers(data)
        } else {
          // Fallback to default brokers if API fails
          setBrokers([
            { id: 'fidelity', name: 'Fidelity' },
            { id: 'schwab', name: 'Charles Schwab' },
            { id: 'etrade', name: 'E*TRADE' },
            { id: 'td-ameritrade', name: 'TD Ameritrade' },
            { id: 'robinhood', name: 'Robinhood' },
            { id: 'interactive-brokers', name: 'Interactive Brokers' },
          ])
        }
      } catch (error) {
        console.error('Failed to fetch brokers:', error)
        // Fallback to default brokers
        setBrokers([
          { id: 'fidelity', name: 'Fidelity' },
          { id: 'schwab', name: 'Charles Schwab' },
          { id: 'etrade', name: 'E*TRADE' },
          { id: 'td-ameritrade', name: 'TD Ameritrade' },
          { id: 'robinhood', name: 'Robinhood' },
          { id: 'interactive-brokers', name: 'Interactive Brokers' },
        ])
      } finally {
        setLoading((prev) => ({ ...prev, brokers: false }))
      }
    }

    const fetchSymbols = async () => {
      setLoading((prev) => ({ ...prev, symbols: true }))
      try {
        // TODO: Replace with actual API endpoint
        const response = await fetch('/api/symbols')
        if (response.ok) {
          const data = await response.json()
          setSymbols(data)
        } else {
          // Fallback to popular symbols if API fails
          setSymbols([
            { symbol: 'AAPL', name: 'Apple Inc.' },
            { symbol: 'GOOGL', name: 'Alphabet Inc.' },
            { symbol: 'MSFT', name: 'Microsoft Corporation' },
            { symbol: 'TSLA', name: 'Tesla, Inc.' },
            { symbol: 'AMZN', name: 'Amazon.com, Inc.' },
            { symbol: 'NVDA', name: 'NVIDIA Corporation' },
          ])
        }
      } catch (error) {
        console.error('Failed to fetch symbols:', error)
        // Fallback to popular symbols
        setSymbols([
          { symbol: 'AAPL', name: 'Apple Inc.' },
          { symbol: 'GOOGL', name: 'Alphabet Inc.' },
          { symbol: 'MSFT', name: 'Microsoft Corporation' },
          { symbol: 'TSLA', name: 'Tesla, Inc.' },
          { symbol: 'AMZN', name: 'Amazon.com, Inc.' },
          { symbol: 'NVDA', name: 'NVIDIA Corporation' },
        ])
      } finally {
        setLoading((prev) => ({ ...prev, symbols: false }))
      }
    }

    fetchCurrencies()
    fetchBrokers()
    fetchSymbols()
  }, [])

  const handleInputChange = useCallback((field: keyof TransactionData, value: string) => {
    setTransaction((prev: TransactionData) => ({
      ...prev,
      [field]: value,
    }))
  }, [])

  // Calculate total amount from price and quantity
  const calculatedAmount = useMemo(() => {
    return transaction.price * transaction.quantity
  }, [transaction.price, transaction.quantity])

  // Format amount display based on trade type
  const formatAmount = useMemo(() => {
    if (calculatedAmount === 0) return '$0.00'

    const formattedValue = calculatedAmount.toFixed(2)

    switch (transaction.trade_type) {
      case 'Sell':
        return `-$${formattedValue}`
      case 'Buy':
      case 'Dividends':
      default:
        return `$${formattedValue}`
    }
  }, [calculatedAmount, transaction.trade_type])

  const addToBatch = useCallback(() => {
    const newTransaction: TransactionData = {
      ...transaction,
      id: `batch-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      amount: calculatedAmount,
    }

    setBatch((prev) => [...prev, newTransaction])

    // Clear form for next transaction
    setTransaction({
      id: '',
      transaction_date: '',
      symbol: '',
      trade_type: TRADE_TYPE.BUY as TradeType,
      price: 0,
      quantity: 0,
      amount: 0,
      broker: '',
      currency: CURRENCY.USD,
      exchange: '',
      user_notes: '',
    })
  }, [transaction, calculatedAmount])

  const handleReview = useCallback(() => {
    // Store batch in localStorage temporarily (or use context/state management)
    localStorage.setItem('transaction-batch', JSON.stringify(batch))
    navigate(ROUTES.TRANSACTIONS_MANUAL_REVIEW)
  }, [navigate, batch])

  const handleCancel = useCallback(() => {
    navigate(ROUTES.TRANSACTIONS_UPLOAD)
  }, [navigate])

  const isFormValid =
    transaction.transaction_date &&
    transaction.symbol &&
    transaction.trade_type &&
    transaction.price &&
    transaction.quantity &&
    transaction.currency

  return (
    <div className="min-h-screen bg-background">
      <main className="container mx-auto py-8 px-4 max-w-2xl">
        {/* Page Title */}
        <div className="text-center mb-8">
          <Title as="h1" className="mb-4">
            Manually Add Transaction
          </Title>
          <p className="text-lg text-muted-foreground">Enter your transaction details below.</p>
        </div>

        <Card>
          <CardContent className="p-8">
            <div className="space-y-6">
              {/* Trade Date */}
              <div>
                <Label htmlFor="trade-date-input">Trade Date *</Label>
                <Input
                  id="trade-date-input"
                  type="date"
                  value={transaction.transaction_date}
                  onChange={(e) => handleInputChange('transaction_date', e.target.value)}
                  className="w-full"
                />
              </div>

              {/* Symbol */}
              <div>
                <Label htmlFor="symbol-dropdown">Symbol *</Label>
                <Dropdown
                  trigger={
                    <DropdownTrigger className="w-full">
                      {transaction.symbol || 'Select or type symbol'}
                    </DropdownTrigger>
                  }
                  className="w-full"
                >
                  {symbols.map((symbol) => (
                    <DropdownItem
                      key={symbol.symbol}
                      onClick={() => handleInputChange('symbol', symbol.symbol)}
                    >
                      {symbol.symbol} - {symbol.name}
                    </DropdownItem>
                  ))}
                  {symbols.length === 0 && (
                    <DropdownItem onClick={() => {}}>
                      {loading.symbols ? 'Loading symbols...' : 'No symbols available'}
                    </DropdownItem>
                  )}
                </Dropdown>
                <Input
                  id="symbol-dropdown"
                  type="text"
                  placeholder="Or type symbol manually (e.g., AAPL, TSLA)"
                  value={transaction.symbol}
                  onChange={(e) => handleInputChange('symbol', e.target.value.toUpperCase())}
                  className="w-full mt-2"
                />
              </div>

              {/* Trade Type */}
              <div>
                <Label htmlFor="trade-type-dropdown">Trade Type *</Label>
                <Dropdown
                  trigger={
                    <DropdownTrigger className="w-full">
                      {transaction.trade_type || 'Select trade type'}
                    </DropdownTrigger>
                  }
                  className="w-full"
                >
                  <DropdownItem onClick={() => handleInputChange('trade_type', 'Buy')}>
                    Buy
                  </DropdownItem>
                  <DropdownItem onClick={() => handleInputChange('trade_type', 'Sell')}>
                    Sell
                  </DropdownItem>
                  <DropdownItem onClick={() => handleInputChange('trade_type', 'Dividends')}>
                    Dividends
                  </DropdownItem>
                </Dropdown>
              </div>

              {/* Price and Quantity Row */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="price-input">Price *</Label>
                  <Input
                    id="price-input"
                    type="number"
                    step="0.01"
                    placeholder="0.00"
                    value={transaction.price}
                    onChange={(e) => handleInputChange('price', e.target.value)}
                    className="w-full"
                  />
                </div>
                <div>
                  <Label htmlFor="quantity-input">Quantity *</Label>
                  <Input
                    id="quantity-input"
                    type="number"
                    step="1"
                    placeholder="0"
                    value={transaction.quantity}
                    onChange={(e) => handleInputChange('quantity', e.target.value)}
                    className="w-full"
                  />
                </div>
              </div>

              {/* Amount */}
              <div>
                <Label>Total Amount</Label>
                <div className="flex h-10 w-full rounded-md border border-input bg-muted/30 px-3 py-2 text-sm text-foreground items-center justify-end">
                  {formatAmount}
                </div>
                <p className="text-xs text-muted-foreground mt-1">
                  Auto-calculated from price × quantity
                </p>
              </div>

              {/* Currency */}
              <div>
                <Label htmlFor="currency-dropdown">Currency *</Label>
                <Dropdown
                  trigger={
                    <DropdownTrigger className="w-full">
                      {transaction.currency || 'Select currency'}
                    </DropdownTrigger>
                  }
                  className="w-full"
                >
                  {currencies.map((currency) => (
                    <DropdownItem
                      key={currency.code}
                      onClick={() => handleInputChange('currency', currency.code)}
                    >
                      {currency.code} - {currency.name}
                    </DropdownItem>
                  ))}
                  {currencies.length === 0 && (
                    <DropdownItem onClick={() => {}}>
                      {loading.currencies ? 'Loading currencies...' : 'No currencies available'}
                    </DropdownItem>
                  )}
                </Dropdown>
              </div>

              {/* Broker */}
              <div>
                <Label htmlFor="broker-dropdown">Broker</Label>
                <Dropdown
                  trigger={
                    <DropdownTrigger className="w-full">
                      {transaction.broker || 'Select broker'}
                    </DropdownTrigger>
                  }
                  className="w-full"
                >
                  {brokers.map((broker) => (
                    <DropdownItem
                      key={broker.id}
                      onClick={() => handleInputChange('broker', broker.name)}
                    >
                      {broker.name}
                    </DropdownItem>
                  ))}
                  {brokers.length === 0 && (
                    <DropdownItem onClick={() => {}}>
                      {loading.brokers ? 'Loading brokers...' : 'No brokers available'}
                    </DropdownItem>
                  )}
                </Dropdown>
                <Input
                  id="broker-dropdown"
                  type="text"
                  placeholder="Or type broker name manually"
                  value={transaction.broker}
                  onChange={(e) => handleInputChange('broker', e.target.value)}
                  className="w-full mt-2"
                />
              </div>

              {/* Notes */}
              <div>
                <Label htmlFor="notes-textarea">Notes</Label>
                <textarea
                  id="notes-textarea"
                  placeholder="Optional notes about this transaction"
                  value={transaction.user_notes}
                  onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
                    handleInputChange('user_notes', e.target.value.slice(0, 100))
                  }
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  rows={3}
                  maxLength={100}
                />
                <p className="text-xs text-muted-foreground mt-1 text-right">
                  {100 - transaction.user_notes.length} characters remaining
                </p>
              </div>
            </div>

            {/* Batch Status Area */}
            {batch.length > 0 && (
              <div className="border-t border-border pt-6 mt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-foreground">
                      {batch.length} trade{batch.length !== 1 ? 's' : ''} added to batch.
                    </p>
                    <button
                      className="text-xs text-muted-foreground hover:text-foreground underline mt-1"
                      onClick={() => {
                        // TODO: Implement a modal or expanded view to show batch details
                        console.log('Current batch:', batch)
                      }}
                    >
                      View all added trades
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* Action Buttons */}
            <div className="flex justify-end gap-4 mt-8">
              <Button variant="secondary" onClick={handleCancel} className="px-6">
                Cancel
              </Button>
              {batch.length > 0 && (
                <Button variant="secondary" onClick={handleReview} className="px-6">
                  Review
                </Button>
              )}
              <Button
                variant="default"
                onClick={addToBatch}
                disabled={!isFormValid}
                className="px-6"
              >
                {batch.length > 0 ? 'Add Another Trade' : 'Add to Batch'}
              </Button>
            </div>
          </CardContent>
        </Card>
      </main>
    </div>
  )
}
