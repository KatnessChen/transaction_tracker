import { useEffect, useState, useCallback, useRef } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { useNavigate } from 'react-router-dom'
import { ROUTES } from '@/constants'
import { TRADE_TYPE, CURRENCY } from '@/constants'
import type { TransactionData } from '@/types'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dropdown, DropdownItem } from '@/components/ui/dropdown'
import DropdownTrigger from '@/components/ui/dropdown-trigger'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
import { EditIcon, DeleteIcon } from '@/components/icons'
import EmptyTransactionsState from '@/components/EmptyTransactionsState'
import { TransactionCard } from '@/components/TransactionCard'
import Title from '@/components/ui/title'
import { TransactionService } from '@/services/transaction.service'
import {
  fetchTransactionHistoryStart,
  fetchTransactionHistorySuccess,
  fetchTransactionHistoryFailure,
} from '@/store/transactionHistorySlice'
import type { GetTransactionHistoryResponse } from '@/store/transactionHistorySlice'
import type { RootState } from '@/store'
import { useToast } from '@/hooks/useToast'

type SortField = keyof TransactionData
type SortDirection = 'asc' | 'desc' | null

export default function TransactionHistoryPage() {
  const navigate = useNavigate()
  const dispatch = useDispatch()
  const { showToast } = useToast()
  const { transactions } = useSelector((state: RootState) => state.transactionHistory)
  const loading = useSelector((state: RootState) => state.transactionHistory.loading)
  const error = useSelector((state: RootState) => state.transactionHistory.error)

  const hasFetchedTransactions = useRef(false)

  const fetchTransactions = useCallback(() => {
    dispatch(fetchTransactionHistoryStart())
    TransactionService.getTransactionHistory()
      .then((data: GetTransactionHistoryResponse) => {
        dispatch(fetchTransactionHistorySuccess(data))
      })
      .catch((err) => {
        dispatch(
          fetchTransactionHistoryFailure(err?.message || 'Failed to fetch transaction history')
        )
        showToast({
          type: 'error',
          title: 'Failed to fetch transactions',
          message: err?.message || 'An error occurred while fetching transaction history.',
        })
      })
  }, [dispatch, showToast])

  useEffect(() => {
    if (hasFetchedTransactions.current) return

    fetchTransactions()

    hasFetchedTransactions.current = true
  }, [fetchTransactions, hasFetchedTransactions])

  const [sortField, setSortField] = useState<SortField>('transaction_date')
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc')

  // Filter states
  const [dateRange, setDateRange] = useState('Last 30 Days')
  const [symbolFilter, setSymbolFilter] = useState('')
  const [currencyFilter, setCurrencyFilter] = useState('All')
  const [brokerFilter, setBrokerFilter] = useState('All')
  const [tradeTypeFilter, setTradeTypeFilter] = useState('All')

  // Pagination states
  const [currentPage, setCurrentPage] = useState(1)
  const itemsPerPage = 10
  const totalPages = Math.ceil(transactions.length / itemsPerPage)

  // Get current page transactions
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const currentTransactions = transactions.slice(startIndex, endIndex)

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('desc')
    }
  }

  const handleApplyFilters = () => {
    // TODO: Implement filter logic
    console.log('Applying filters:', {
      dateRange,
      symbolFilter,
      currencyFilter,
      brokerFilter,
      tradeTypeFilter,
    })
  }

  const handleClearFilters = () => {
    setDateRange('Last 30 Days')
    setSymbolFilter('')
    setCurrencyFilter('All')
    setBrokerFilter('All')
    setTradeTypeFilter('All')
  }

  const handleEditTransaction = (transaction: TransactionData) => {
    // TODO: Implement edit transaction modal/form
    console.log('Editing transaction:', transaction)
  }

  const handleDeleteTransaction = (id: string) => {
    // TODO: Implement delete confirmation and API call
    console.log('Deleting transaction:', id)
  }

  const handleUpdateNotes = (id: string, notes: string) => {
    // TODO: Implement API call to update notes
    console.log('Updating notes for transaction:', id, notes)
  }

  // Optionally show loading and error states in the UI
  if (loading) {
    return (
      <div className="container mx-auto py-6 px-4">
        <Title as="h1">Transaction History</Title>
        <div className="flex justify-center items-center h-64">
          <span className="text-muted-foreground">Loading transactions...</span>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="container mx-auto py-6 px-4">
        <Title as="h1">Transaction History</Title>
        <div className="flex justify-center items-center h-64">
          <span className="text-destructive">{error}</span>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto py-6 px-4 space-y-8">
      {/* Page Title */}
      <Title as="h1">Transaction History</Title>

      {transactions.length === 0 ? (
        <EmptyTransactionsState onUploadClick={() => navigate(ROUTES.TRANSACTIONS_UPLOAD)} />
      ) : (
        <>
          {/* Filters Section */}
          <section className="space-y-3">
            <div className="flex flex-wrap gap-3 mb-4">
              {/* Date Range Filter */}
              <div>
                <Label htmlFor="date-range" className="text-sm text-muted-foreground">
                  Date Range
                </Label>
                <Dropdown
                  trigger={
                    <DropdownTrigger className="w-full justify-between border border-input">
                      <span className="truncate">{dateRange}</span>
                    </DropdownTrigger>
                  }
                >
                  <DropdownItem onClick={() => setDateRange('Last 7 Days')}>
                    Last 7 Days
                  </DropdownItem>
                  <DropdownItem onClick={() => setDateRange('Last 30 Days')}>
                    Last 30 Days
                  </DropdownItem>
                  <DropdownItem onClick={() => setDateRange('Last 3 Months')}>
                    Last 3 Months
                  </DropdownItem>
                  <DropdownItem onClick={() => setDateRange('Last Year')}>Last Year</DropdownItem>
                </Dropdown>
              </div>

              {/* Symbol Filter */}
              <div>
                <Label htmlFor="symbol" className="text-sm text-muted-foreground">
                  Symbol
                </Label>
                <Input
                  id="symbol"
                  placeholder="e.g. AAPL"
                  className="w-full"
                  value={symbolFilter}
                  onChange={(e) => setSymbolFilter(e.target.value)}
                />
              </div>

              {/* Currency Filter */}
              <div>
                <Label htmlFor="currency" className="text-sm text-muted-foreground">
                  Currency
                </Label>
                <Dropdown
                  trigger={
                    <DropdownTrigger className="w-full justify-between border border-input">
                      <span className="truncate">{currencyFilter}</span>
                    </DropdownTrigger>
                  }
                >
                  <DropdownItem onClick={() => setCurrencyFilter('All')}>All</DropdownItem>
                  <DropdownItem onClick={() => setCurrencyFilter(CURRENCY.USD)}>
                    {CURRENCY.USD}
                  </DropdownItem>
                  <DropdownItem onClick={() => setCurrencyFilter(CURRENCY.TWD)}>
                    {CURRENCY.TWD}
                  </DropdownItem>
                  <DropdownItem onClick={() => setCurrencyFilter(CURRENCY.CAD)}>
                    {CURRENCY.CAD}
                  </DropdownItem>
                </Dropdown>
              </div>

              {/* Broker Filter */}
              <div>
                <Label htmlFor="broker" className="text-sm text-muted-foreground">
                  Broker
                </Label>
                <Dropdown
                  trigger={
                    <DropdownTrigger className="w-full justify-between border border-input">
                      <span className="truncate">{brokerFilter}</span>
                    </DropdownTrigger>
                  }
                >
                  <DropdownItem onClick={() => setBrokerFilter('All')}>All</DropdownItem>
                  <DropdownItem onClick={() => setBrokerFilter('Fidelity')}>Fidelity</DropdownItem>
                  <DropdownItem onClick={() => setBrokerFilter('Schwab')}>Schwab</DropdownItem>
                  <DropdownItem onClick={() => setBrokerFilter('E*TRADE')}>E*TRADE</DropdownItem>
                  <DropdownItem onClick={() => setBrokerFilter('TD Ameritrade')}>
                    TD Ameritrade
                  </DropdownItem>
                </Dropdown>
              </div>

              {/* Trade Type Filter */}
              <div>
                <Label htmlFor="trade-type" className="text-sm text-muted-foreground">
                  Trade Type
                </Label>
                <Dropdown
                  trigger={
                    <DropdownTrigger className="w-full justify-between border border-input">
                      <span className="truncate">{tradeTypeFilter}</span>
                    </DropdownTrigger>
                  }
                >
                  <DropdownItem onClick={() => setTradeTypeFilter('All')}>All</DropdownItem>
                  <DropdownItem onClick={() => setTradeTypeFilter(TRADE_TYPE.BUY)}>
                    {TRADE_TYPE.BUY}
                  </DropdownItem>
                  <DropdownItem onClick={() => setTradeTypeFilter(TRADE_TYPE.SELL)}>
                    {TRADE_TYPE.SELL}
                  </DropdownItem>
                  <DropdownItem onClick={() => setTradeTypeFilter(TRADE_TYPE.DIVIDEND)}>
                    {TRADE_TYPE.DIVIDEND}
                  </DropdownItem>
                </Dropdown>
              </div>
            </div>

            {/* Filter Action Buttons */}
            <div className="flex gap-3">
              <Button variant="default" onClick={handleApplyFilters}>
                Apply Filters
              </Button>
              <Button variant="secondary" onClick={handleClearFilters}>
                Clear Filters
              </Button>
            </div>
          </section>

          {/* Transaction History Table Section */}
          <section className="space-y-4">
            {/* Desktop Table View - Hidden on mobile */}
            <div className="hidden md:block">
              <Card className="bg-card">
                <CardContent className="p-0">
                  <div className="rounded-md overflow-hidden">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead
                            sortable
                            sortDirection={sortField === 'transaction_date' ? sortDirection : null}
                            onSort={() => handleSort('transaction_date')}
                            className="w-[120px]"
                          >
                            Trade Date
                          </TableHead>
                          <TableHead
                            sortable
                            sortDirection={sortField === 'symbol' ? sortDirection : null}
                            onSort={() => handleSort('symbol')}
                          >
                            Symbol
                          </TableHead>
                          <TableHead
                            sortable
                            sortDirection={sortField === 'trade_type' ? sortDirection : null}
                            onSort={() => handleSort('trade_type')}
                          >
                            Trade Type
                          </TableHead>
                          <TableHead
                            sortable
                            sortDirection={sortField === 'price' ? sortDirection : null}
                            onSort={() => handleSort('price')}
                          >
                            Price
                          </TableHead>
                          <TableHead
                            sortable
                            sortDirection={sortField === 'quantity' ? sortDirection : null}
                            onSort={() => handleSort('quantity')}
                          >
                            Quantity
                          </TableHead>
                          <TableHead
                            sortable
                            sortDirection={sortField === 'amount' ? sortDirection : null}
                            onSort={() => handleSort('amount')}
                            className="text-right"
                          >
                            Amount
                          </TableHead>
                          <TableHead
                            sortable
                            sortDirection={sortField === 'currency' ? sortDirection : null}
                            onSort={() => handleSort('currency')}
                          >
                            Currency
                          </TableHead>
                          <TableHead
                            sortable
                            sortDirection={sortField === 'broker' ? sortDirection : null}
                            onSort={() => handleSort('broker')}
                          >
                            Broker
                          </TableHead>
                          <TableHead
                            sortable
                            sortDirection={sortField === 'user_notes' ? sortDirection : null}
                            onSort={() => handleSort('user_notes')}
                          >
                            Notes
                          </TableHead>
                          <TableHead>Action</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {currentTransactions.map((transaction) => (
                          <TableRow key={transaction.id}>
                            <TableCell>{transaction.transaction_date}</TableCell>
                            <TableCell className="font-medium">{transaction.symbol}</TableCell>
                            <TableCell
                              className={`font-medium ${
                                transaction.trade_type === TRADE_TYPE.BUY
                                  ? 'text-primary'
                                  : transaction.trade_type === TRADE_TYPE.SELL
                                    ? 'text-chart-1'
                                    : 'text-muted'
                              }`}
                            >
                              {transaction.trade_type}
                            </TableCell>
                            <TableCell className="text-right text-muted-foreground">
                              ${transaction.price}
                            </TableCell>
                            <TableCell className="text-right text-muted-foreground">
                              {transaction.quantity}
                            </TableCell>
                            <TableCell className="text-right">
                              {transaction.trade_type === TRADE_TYPE.SELL ? '-' : ''}$
                              {transaction.amount}
                            </TableCell>
                            <TableCell className="text-right">{transaction.currency}</TableCell>
                            <TableCell>{transaction.broker}</TableCell>
                            <TableCell className="max-w-32 truncate">
                              {transaction.user_notes}
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-2">
                                <button
                                  className="p-1 hover:bg-muted rounded transition-colors"
                                  onClick={() => handleEditTransaction(transaction)}
                                  title="Edit transaction"
                                >
                                  <EditIcon
                                    size={16}
                                    className="text-muted-foreground hover:text-primary"
                                  />
                                </button>
                                <button
                                  className="p-1 hover:bg-muted rounded transition-colors"
                                  onClick={() => handleDeleteTransaction(transaction.id)}
                                  title="Delete transaction"
                                >
                                  <DeleteIcon
                                    size={16}
                                    className="text-muted-foreground hover:text-destructive"
                                  />
                                </button>
                              </div>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Mobile Cards View - Visible only on mobile */}
            <div className="md:hidden space-y-3">
              {currentTransactions.map((transaction) => (
                <TransactionCard
                  key={transaction.id}
                  transaction={transaction}
                  onEdit={handleEditTransaction}
                  onDelete={handleDeleteTransaction}
                  onUpdateNotes={handleUpdateNotes}
                />
              ))}
            </div>

            {/* Pagination */}
            <Pagination>
              <PaginationContent>
                <PaginationItem>
                  <PaginationPrevious
                    href="#"
                    onClick={(e) => {
                      e.preventDefault()
                      if (currentPage > 1) setCurrentPage(currentPage - 1)
                    }}
                    style={{
                      opacity: currentPage === 1 ? 0.5 : 1,
                      pointerEvents: currentPage === 1 ? 'none' : 'auto',
                    }}
                  />
                </PaginationItem>

                {/* Generate page numbers */}
                {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => {
                  const pageNumber = i + 1
                  return (
                    <PaginationItem key={pageNumber}>
                      <PaginationLink
                        href="#"
                        isActive={pageNumber === currentPage}
                        onClick={(e) => {
                          e.preventDefault()
                          setCurrentPage(pageNumber)
                        }}
                      >
                        {pageNumber}
                      </PaginationLink>
                    </PaginationItem>
                  )
                })}

                {totalPages > 5 && (
                  <PaginationItem>
                    <PaginationEllipsis />
                  </PaginationItem>
                )}

                <PaginationItem>
                  <PaginationNext
                    href="#"
                    onClick={(e) => {
                      e.preventDefault()
                      if (currentPage < totalPages) setCurrentPage(currentPage + 1)
                    }}
                    style={{
                      opacity: currentPage === totalPages ? 0.5 : 1,
                      pointerEvents: currentPage === totalPages ? 'none' : 'auto',
                    }}
                  />
                </PaginationItem>
              </PaginationContent>
            </Pagination>
          </section>
        </>
      )}
    </div>
  )
}
