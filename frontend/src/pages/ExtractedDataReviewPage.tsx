import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSelector, useDispatch } from 'react-redux'
import { ROUTES } from '@/constants'
import { CURRENCY } from '@/constants/setting'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ConfirmationModal, ImageViewerModal } from '@/components/ui'
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
import { DeleteIcon, PlusIcon, ClockIcon, SpinnerIcon, CheckIcon, XIcon } from '@/components/icons'
import { Title } from '@/components/ui/title'
import type { RootState } from '@/store'
import type { TransactionData, TradeType } from '@/types'
import {
  FILE_STATUS_PROCESSING,
  FILE_STATUS_COMPLETED,
  FILE_STATUS_ERROR,
} from '@/constants/fileProcessingStatus'
import {
  updateExtractedTransactions,
  clearValidationError,
  clearCurrentFile,
} from '@/store/fileProcessingSlice'
import { getSerializableFileUrl, getFileStatus, getStatusIconName } from '@/utils/fileUtils'
import { TransactionService } from '@/services/transaction.service'
import { useToast } from '@/hooks/useToast'
import { abs } from '@/utils'
import {
  validateTransactionDate,
  validateTransactionQuantity,
  validateTransactionPrice,
} from '@/utils/transactionValidation'

const statusIconMap = {
  ClockIcon,
  SpinnerIcon,
  CheckIcon,
  XIcon,
}

export default function ExtractedDataReviewPage() {
  const navigate = useNavigate()
  const dispatch = useDispatch()
  const { showToast } = useToast()

  const { files, fileStates, extractResults, validationErrors } = useSelector(
    (state: RootState) => state.fileProcessing
  )

  const [currentFileIndex, setCurrentFileIndex] = useState(-1)
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set())
  const [showDeleteConfirmation, setShowDeleteConfirmation] = useState(false)
  const [showCancelConfirmation, setShowCancelConfirmation] = useState(false)
  const [isImporting, setIsImporting] = useState(false)
  const [showImageViewer, setShowImageViewer] = useState(false)

  useEffect(() => {
    // Redirect if no files in store
    if (files.length === 0) {
      navigate(ROUTES.TRANSACTIONS_UPLOAD)
      return
    }

    // Find the first available (completed) file index
    const firstAvailableIndex = fileStates.findIndex((fs) => fs.status === 'completed')
    setCurrentFileIndex(firstAvailableIndex >= 0 ? firstAvailableIndex : 0)
  }, [navigate, files, fileStates])

  // Helper functions
  const getTransactionCount = (fileIndex: number) => {
    if (!extractResults[fileIndex]) return 0
    return extractResults[fileIndex].transaction_count
  }

  const areAllFilesProcessed = () => {
    return fileStates.every(
      (fs) => fs.status === FILE_STATUS_COMPLETED || fs.status === FILE_STATUS_ERROR
    )
  }

  const hasValidationErrors = () => {
    const currentFilePrefix = `file-${currentFileIndex}-`
    return Object.keys(validationErrors).some((key) => key.startsWith(currentFilePrefix))
  }

  const findNextCompletedFile = (excludeIndex: number) => {
    return fileStates.findIndex(
      (fs, index) => index !== excludeIndex && fs.status === FILE_STATUS_COMPLETED
    )
  }

  // Get transactions for current file (use Redux data as source of truth)
  const getCurrentFileTransactions = (): TransactionData[] => {
    if (!extractResults[currentFileIndex]) return []

    const fileResult = extractResults[currentFileIndex]
    return fileResult.transactions.map((transaction, index) => ({
      id: `${currentFileIndex}-${index}`,
      symbol: transaction.symbol,
      trade_type: transaction.trade_type,
      quantity: transaction.quantity,
      price: transaction.price,
      amount: transaction.amount,
      transaction_date: transaction.transaction_date,
      broker: transaction.broker,
      currency: transaction.currency,
      user_notes: transaction.user_notes,
      exchange: transaction.exchange,
    }))
  }

  const currentTransactions = getCurrentFileTransactions()

  const handleCellEdit = (id: string, field: keyof TransactionData, value: string | number) => {
    const errorKey = `file-${currentFileIndex}-${id}-${field}`

    // Clear previous error for this field
    dispatch(clearValidationError({ errorKey }))

    // Always update the value first, then validate
    if (currentFileIndex >= 0 && extractResults[currentFileIndex]) {
      // Get current transactions for this file from Redux
      const currentFileTransactions = getCurrentFileTransactions().map((t) => {
        if (t.id === id) {
          const updatedTransaction = { ...t, [field]: value }

          // Recalculate amount if quantity or price changes (but not for Dividends)
          if ((field === 'quantity' || field === 'price') && t.trade_type !== 'Dividends') {
            const quantity = field === 'quantity' ? (value as number) : t.quantity
            const price = field === 'price' ? (value as number) : t.price
            updatedTransaction.amount = abs(quantity * price)
          }

          return updatedTransaction
        }
        return t
      })

      // Convert to Redux format (remove the id and use original format)
      const reduxTransactions = currentFileTransactions.map((t) => ({
        id: t.id,
        symbol: t.symbol,
        trade_type: t.trade_type,
        quantity: t.quantity,
        price: t.price,
        amount: t.amount,
        transaction_date: t.transaction_date,
        broker: t.broker,
        currency: t.currency,
        user_notes: t.user_notes,
        exchange: t.exchange,
      }))

      // Update Redux store
      dispatch(
        updateExtractedTransactions({
          fileIndex: currentFileIndex,
          transactions: reduxTransactions,
        })
      )
    }

    // Validation rules (after updating the value)
    if (field === 'transaction_date') {
      if (!validateTransactionDate(value as string, errorKey, dispatch)) return
    }

    if (field === 'quantity' || field === 'price') {
      const numericValue = value as number
      // Find the trade type for this transaction
      const tradeType = currentTransactions.find((t) => t.id === id)?.trade_type || ''
      if (field === 'quantity') {
        if (!validateTransactionQuantity(tradeType, numericValue, errorKey, dispatch)) return
      }
      if (field === 'price') {
        if (!validateTransactionPrice(tradeType, numericValue, errorKey, dispatch)) return
      }
    }
  }

  const handleAddRow = () => {
    if (currentFileIndex >= 0 && extractResults[currentFileIndex]) {
      const newTransaction = {
        id: `${currentFileIndex}-${extractResults[currentFileIndex].transactions.length}`,
        symbol: '',
        trade_type: 'Buy' as TradeType,
        quantity: 0,
        price: 0,
        amount: 0,
        transaction_date: new Date().toISOString().split('T')[0],
        broker: '',
        currency: 'USD',
        user_notes: '',
        account: '',
        exchange: '',
      }
      const currentFileTransactions = [
        ...extractResults[currentFileIndex].transactions,
        newTransaction,
      ]
      dispatch(
        updateExtractedTransactions({
          fileIndex: currentFileIndex,
          transactions: currentFileTransactions,
        })
      )
    }
  }

  const handleDeleteRows = () => {
    setShowDeleteConfirmation(true)
  }

  const confirmDeleteRows = () => {
    if (currentFileIndex >= 0 && extractResults[currentFileIndex]) {
      const currentFileTransactions = extractResults[currentFileIndex].transactions.filter(
        (_, idx) => {
          const transactionId = `${currentFileIndex}-${idx}`
          return !selectedRows.has(transactionId)
        }
      )
      dispatch(
        updateExtractedTransactions({
          fileIndex: currentFileIndex,
          transactions: currentFileTransactions,
        })
      )
    }
    setSelectedRows(new Set())
    setShowDeleteConfirmation(false)
  }

  const cancelDeleteRows = () => {
    setShowDeleteConfirmation(false)
  }

  const handleRowSelect = (id: string) => {
    const newSelected = new Set(selectedRows)
    if (newSelected.has(id)) {
      newSelected.delete(id)
    } else {
      newSelected.add(id)
    }
    setSelectedRows(newSelected)
  }

  const handleConfirmImport = async () => {
    if (currentFileIndex < 0 || !extractResults[currentFileIndex]) {
      return
    }

    setIsImporting(true)

    try {
      // Get current transactions and prepare them for API call
      const transactionsToImport = extractResults[currentFileIndex].transactions

      console.log('Importing transactions:', transactionsToImport)

      // Send to backend
      const result = await TransactionService.importTransactions(transactionsToImport)

      console.log('Import result:', result)

      // Show success toast
      showToast({
        type: 'success',
        title: 'Import Successful',
        message: `${result.data.count || transactionsToImport.length} transactions imported successfully!`,
        duration: 5000,
      })

      // Clear current file after successful import
      dispatch(clearCurrentFile({ fileIndex: currentFileIndex }))

      // Find next completed file
      const nextFileIndex = findNextCompletedFile(currentFileIndex)

      if (nextFileIndex >= 0) {
        // Adjust index if it's after the removed file
        const adjustedIndex = nextFileIndex > currentFileIndex ? nextFileIndex - 1 : nextFileIndex
        setCurrentFileIndex(adjustedIndex)
      } else {
        // No more files, navigate to transactions page with final message
        navigate(ROUTES.TRANSACTIONS, {
          state: {
            message: `All files processed! ${result.data.count} transactions imported successfully.`,
          },
        })
      }
    } catch (error) {
      showToast({
        type: 'error',
        title: 'Import Failed',
        message:
          error instanceof Error
            ? error.message
            : 'Failed to import transactions. Please try again.',
        duration: 8000, // Show error longer
      })
    } finally {
      setIsImporting(false)
    }
  }

  const handleCancel = () => {
    setShowCancelConfirmation(true)
  }

  const confirmCancel = () => {
    // Clear current file and its data
    dispatch(clearCurrentFile({ fileIndex: currentFileIndex }))

    // Find next completed file
    const nextFileIndex = findNextCompletedFile(currentFileIndex)

    if (nextFileIndex >= 0) {
      // Adjust index if it's after the removed file
      const adjustedIndex = nextFileIndex > currentFileIndex ? nextFileIndex - 1 : nextFileIndex
      setCurrentFileIndex(adjustedIndex)
    } else {
      // No more completed files, navigate to upload page
      navigate(ROUTES.TRANSACTIONS_UPLOAD)
    }

    setShowCancelConfirmation(false)
  }

  const cancelCancel = () => {
    setShowCancelConfirmation(false)
  }

  return (
    <div>
      <div className="container mx-auto p-6 space-y-6">
        {/* Page Title */}
        <div className="text-center mb-8">
          <Title as="h1" className="mb-4">
            Review & Confirm Transactions
          </Title>
          <p className="text-muted-foreground">
            {areAllFilesProcessed()
              ? 'All files processed. Please review the extracted data below and make any necessary corrections.'
              : 'Some data is ready, other files are still processing. You can start reviewing now.'}
          </p>
        </div>

        {/* File tabs */}
        <div className="flex flex-wrap gap-2 mb-6">
          {files.map((file, index) => {
            const status = getFileStatus(fileStates, index)
            const count = getTransactionCount(index)
            const isActive = index === currentFileIndex
            const isClickable = status === FILE_STATUS_COMPLETED || status === FILE_STATUS_ERROR
            const IconComponent = statusIconMap[getStatusIconName(status)]

            return (
              <Button
                key={index}
                onClick={() => isClickable && setCurrentFileIndex(index)}
                disabled={!isClickable}
                variant={isActive ? 'default' : 'outline'}
                size="sm"
              >
                <div className="flex items-center gap-2">
                  <span>{IconComponent && <IconComponent size={24} />}</span>
                  <span className="truncate max-w-[150px]">{file.name}</span>
                  {status === FILE_STATUS_COMPLETED && count > 0 && (
                    <span className="text-xs">({count} records)</span>
                  )}
                  {status === FILE_STATUS_PROCESSING && (
                    <span className="text-xs">Processing...</span>
                  )}
                  {status === FILE_STATUS_ERROR && (
                    <span className="text-xs text-red-500">Error</span>
                  )}
                </div>
              </Button>
            )
          })}
        </div>

        {/* Content area */}
        <div className="grid grid-cols-1 gap-6">
          {/* Screenshot display */}
          <Card>
            <CardContent>
              <h3 className="text-lg font-semibold mb-4">Original Screenshot</h3>
              {files[currentFileIndex] && (
                <div className="bg-muted rounded-lg p-4 text-center">
                  <img
                    src={getSerializableFileUrl(files[currentFileIndex])}
                    alt={files[currentFileIndex].name}
                    className="max-w-full h-auto rounded-lg cursor-pointer hover:opacity-80 transition-opacity"
                    onClick={() => setShowImageViewer(true)}
                    title="Click to view in full screen"
                  />
                </div>
              )}
            </CardContent>
          </Card>

          {/* Transaction data */}
          <Card>
            <CardContent>
              <div className="flex justify-between items-center mb-4">
                <h3 className="text-lg font-semibold">Extracted Data</h3>
                <div className="flex gap-2">
                  <Button onClick={handleAddRow} variant="outline" size="sm">
                    <PlusIcon className="w-4 h-4 mr-2" />
                    Add Row
                  </Button>
                  {selectedRows.size > 0 && (
                    <Button
                      onClick={handleDeleteRows}
                      variant="outline"
                      size="sm"
                      className="text-red-600 hover:text-red-700"
                    >
                      <DeleteIcon className="w-4 h-4 mr-2" />
                      Delete Selected ({selectedRows.size})
                    </Button>
                  )}
                </div>
              </div>

              {currentTransactions.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  {getFileStatus(fileStates, currentFileIndex) === FILE_STATUS_PROCESSING
                    ? 'Processing file...'
                    : getFileStatus(fileStates, currentFileIndex) === FILE_STATUS_ERROR
                      ? 'Error processing this file'
                      : 'No transactions found in this file'}
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="w-[50px]">Select</TableHead>
                        <TableHead>Date</TableHead>
                        <TableHead>Symbol</TableHead>
                        <TableHead>Trade Type</TableHead>
                        <TableHead>Quantity</TableHead>
                        <TableHead>Price</TableHead>
                        <TableHead>Amount</TableHead>
                        <TableHead>Broker</TableHead>
                        <TableHead>Exchange</TableHead>
                        <TableHead>Currency</TableHead>
                        <TableHead>Notes</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {currentTransactions.map((transaction) => {
                        return (
                          <TableRow key={transaction.id}>
                            <TableCell>
                              <input
                                type="checkbox"
                                checked={selectedRows.has(transaction.id)}
                                onChange={() => handleRowSelect(transaction.id)}
                                className="rounded"
                              />
                            </TableCell>
                            <TableCell>
                              <div className="relative">
                                <Input
                                  type="date"
                                  value={transaction.transaction_date}
                                  onChange={(e) =>
                                    handleCellEdit(
                                      transaction.id,
                                      'transaction_date',
                                      e.target.value
                                    )
                                  }
                                  className={`w-[120px]`}
                                />
                                {validationErrors[
                                  `file-${currentFileIndex}-${transaction.id}-transaction_date`
                                ] && (
                                  <div className="absolute top-8 left-0 z-20 bg-red-50 border border-red-200 rounded-md p-1 shadow-lg w-[250px]">
                                    <p className="text-xs text-red-700 font-medium">
                                      {
                                        validationErrors[
                                          `file-${currentFileIndex}-${transaction.id}-transaction_date`
                                        ]
                                      }
                                    </p>
                                  </div>
                                )}
                              </div>
                            </TableCell>
                            <TableCell>
                              <Input
                                value={transaction.symbol}
                                onChange={(e) =>
                                  handleCellEdit(transaction.id, 'symbol', e.target.value)
                                }
                                className="w-[100px]"
                              />
                            </TableCell>
                            <TableCell>
                              <Dropdown
                                trigger={
                                  <DropdownTrigger>{transaction.trade_type}</DropdownTrigger>
                                }
                              >
                                <DropdownItem
                                  onClick={() =>
                                    handleCellEdit(transaction.id, 'trade_type', 'Buy')
                                  }
                                >
                                  Buy
                                </DropdownItem>
                                <DropdownItem
                                  onClick={() =>
                                    handleCellEdit(transaction.id, 'trade_type', 'Sell')
                                  }
                                >
                                  Sell
                                </DropdownItem>
                                <DropdownItem
                                  onClick={() =>
                                    handleCellEdit(transaction.id, 'trade_type', 'Dividends')
                                  }
                                >
                                  Dividends
                                </DropdownItem>
                              </Dropdown>
                            </TableCell>
                            <TableCell>
                              <div className="relative">
                                <Input
                                  type="number"
                                  value={transaction.quantity}
                                  onChange={(e) =>
                                    handleCellEdit(
                                      transaction.id,
                                      'quantity',
                                      parseFloat(e.target.value) || 0
                                    )
                                  }
                                  className={`w-[100px] ${
                                    validationErrors[
                                      `file-${currentFileIndex}-${transaction.id}-quantity`
                                    ]
                                      ? 'border-red-500 focus:border-red-500 focus-visible:ring-red-500'
                                      : ''
                                  }`}
                                />
                                {validationErrors[
                                  `file-${currentFileIndex}-${transaction.id}-quantity`
                                ] && (
                                  <div className="absolute top-8 left-0 z-20 bg-red-50 border border-red-200 rounded-md p-1 shadow-lg w-[120px]">
                                    <p className="text-xs text-red-700 font-medium">
                                      {
                                        validationErrors[
                                          `file-${currentFileIndex}-${transaction.id}-quantity`
                                        ]
                                      }
                                    </p>
                                  </div>
                                )}
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="relative">
                                <Input
                                  type="number"
                                  step="0.01"
                                  value={transaction.price}
                                  onChange={(e) =>
                                    handleCellEdit(
                                      transaction.id,
                                      'price',
                                      parseFloat(e.target.value) || 0
                                    )
                                  }
                                  className={`w-[100px] ${
                                    validationErrors[
                                      `file-${currentFileIndex}-${transaction.id}-price`
                                    ]
                                      ? 'border-red-500 focus:border-red-500 focus-visible:ring-red-500'
                                      : ''
                                  }`}
                                />
                                {validationErrors[
                                  `file-${currentFileIndex}-${transaction.id}-price`
                                ] && (
                                  <div className="absolute top-8 left-0 z-20 bg-red-50 border border-red-200 rounded-md p-1 shadow-lg w-[100px]">
                                    <p className="text-xs text-red-700 font-medium">
                                      {
                                        validationErrors[
                                          `file-${currentFileIndex}-${transaction.id}-price`
                                        ]
                                      }
                                    </p>
                                  </div>
                                )}
                              </div>
                            </TableCell>
                            <TableCell>
                              {transaction.trade_type === 'Dividends' ? (
                                <Input
                                  type="number"
                                  step="0.01"
                                  value={transaction.amount}
                                  onChange={(e) =>
                                    handleCellEdit(
                                      transaction.id,
                                      'amount',
                                      parseFloat(e.target.value) || 0
                                    )
                                  }
                                  className="w-[120px]"
                                />
                              ) : (
                                <span className={'font-medium'}>
                                  ${transaction.amount.toFixed(2)}
                                </span>
                              )}
                            </TableCell>
                            <TableCell>
                              <Input
                                value={transaction.broker}
                                onChange={(e) =>
                                  handleCellEdit(transaction.id, 'broker', e.target.value)
                                }
                                className="w-[100px]"
                              />
                            </TableCell>
                            <TableCell>
                              <Input
                                value={transaction.exchange}
                                onChange={(e) =>
                                  handleCellEdit(transaction.id, 'exchange', e.target.value)
                                }
                                className="w-[100px]"
                              />
                            </TableCell>
                            <TableCell>
                              <Dropdown
                                trigger={<DropdownTrigger>{transaction.currency}</DropdownTrigger>}
                              >
                                {Object.entries(CURRENCY).map(([key, value]) => (
                                  <DropdownItem
                                    key={key}
                                    onClick={() =>
                                      handleCellEdit(transaction.id, 'currency', value)
                                    }
                                  >
                                    <span className="flex items-center gap-2">{value}</span>
                                  </DropdownItem>
                                ))}
                              </Dropdown>
                            </TableCell>
                            <TableCell>
                              <Input
                                value={transaction.user_notes}
                                onChange={(e) =>
                                  handleCellEdit(transaction.id, 'user_notes', e.target.value)
                                }
                                className="w-[150px]"
                                placeholder="Add notes..."
                              />
                            </TableCell>
                          </TableRow>
                        )
                      })}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Action buttons */}
        <div className="flex justify-between">
          <Button onClick={handleCancel} variant="outline">
            Discard Current File
          </Button>
          {getTransactionCount(currentFileIndex) > 0 && (
            <Button
              onClick={handleConfirmImport}
              disabled={!areAllFilesProcessed() || hasValidationErrors() || isImporting}
              className="bg-green-600 hover:bg-green-700 text-white"
            >
              {isImporting
                ? 'Importing...'
                : hasValidationErrors()
                  ? 'Fix validation errors first'
                  : areAllFilesProcessed()
                    ? 'Confirm & Import Current File'
                    : 'Import (waiting for all files to complete)'}
            </Button>
          )}
        </div>
      </div>
      {/* Delete confirmation modal */}
      <ConfirmationModal
        isOpen={showDeleteConfirmation}
        title="Confirm Deletion"
        message="Are you sure you want to delete the selected rows? This action cannot be undone."
        confirmText="Delete"
        cancelText="Cancel"
        confirmVariant="destructive"
        onConfirm={confirmDeleteRows}
        onCancel={cancelDeleteRows}
      />

      {/* Cancel confirmation modal */}
      <ConfirmationModal
        isOpen={showCancelConfirmation}
        title="Discard Current File"
        message={`Are you sure you want to discard this file and all its extracted data? This action cannot be undone.${
          findNextCompletedFile(currentFileIndex) >= 0
            ? ' You will be redirected to the next available file.'
            : ' You will be redirected to the upload page.'
        }`}
        confirmText="Discard File"
        cancelText="Keep File"
        confirmVariant="destructive"
        onConfirm={confirmCancel}
        onCancel={cancelCancel}
      />

      {/* Image viewer modal */}
      {files[currentFileIndex] && (
        <ImageViewerModal
          isOpen={showImageViewer}
          imageUrl={getSerializableFileUrl(files[currentFileIndex])}
          imageName={files[currentFileIndex].name}
          onClose={() => setShowImageViewer(false)}
        />
      )}
    </div>
  )
}
