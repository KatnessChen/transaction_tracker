import React, { useState } from 'react'
import { Card, CardContent } from './ui/card'
import { Input } from './ui/input'
import { EditIcon } from './icons/EditIcon'
import { DeleteIcon } from './icons/DeleteIcon'
import type { TransactionData } from '@/types'
import { TRADE_TYPE } from '../constants'

interface TransactionCardProps {
  transaction: TransactionData
  onEdit: (transaction: TransactionData) => void
  onDelete: (id: string) => void
  onUpdateNotes?: (id: string, notes: string) => void
}

export function TransactionCard({
  transaction,
  onEdit,
  onDelete,
  onUpdateNotes,
}: TransactionCardProps) {
  const [isExpanded, setIsExpanded] = useState(false)
  const [isEditingNotes, setIsEditingNotes] = useState(false)
  const [notes, setNotes] = useState(transaction.user_notes || '')

  const handleCardClick = (e: React.MouseEvent) => {
    // Don't toggle if clicking on action buttons or input field
    if (
      (e.target as HTMLElement).closest('button') ||
      (e.target as HTMLElement).closest('input') ||
      (e.target as HTMLElement).closest('[data-action]')
    ) {
      return
    }
    setIsExpanded(!isExpanded)
  }

  const handleNotesSubmit = () => {
    if (onUpdateNotes) {
      onUpdateNotes(transaction.id, notes)
    }
    setIsEditingNotes(false)
  }

  const handleNotesKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleNotesSubmit()
    } else if (e.key === 'Escape') {
      setNotes(transaction.user_notes || '')
      setIsEditingNotes(false)
    }
  }

  // Trade Type colors based on wireframe
  const getTradeTypeColorClass = (tradeType: string) => {
    switch (tradeType.toUpperCase()) {
      case TRADE_TYPE.BUY:
        return 'text-primary' // Brighter Sage Green
      case TRADE_TYPE.SELL:
        return 'text-chart-1' // Brighter Soft Salmon Pink
      case TRADE_TYPE.DIVIDEND:
        return 'text-muted' // Medium Grey-Green
      default:
        return 'text-foreground' // Pure Light Gray fallback
    }
  }

  // Amount formatting with sign
  const formatAmount = (amount: number, tradeType: string) => {
    const sign = tradeType.toUpperCase() === TRADE_TYPE.SELL ? '-' : ''
    return `${sign}${Math.abs(amount).toFixed(2)}`
  }

  return (
    <Card
      className="bg-card border-none rounded-lg cursor-pointer transition-all duration-200 ease-in-out hover:bg-card/90 hover:shadow-lg hover:scale-[1.01] active:scale-[0.99] active:transition-transform active:duration-100"
      onClick={handleCardClick}
    >
      <CardContent className="p-4">
        {/* Row 1: Symbol + Trade Type */}
        <div className="flex justify-between items-center mb-2 transition-all duration-200 hover:translate-x-1">
          <span className="text-foreground font-semibold text-lg">{transaction.symbol}</span>
          <span
            className={`font-medium text-sm transition-all duration-200 ${getTradeTypeColorClass(transaction.trade_type)}`}
          >
            {transaction.trade_type}
          </span>
        </div>

        {/* Row 2: Trade Date + Amount */}
        <div className="flex justify-between items-center mb-2 transition-all duration-200 hover:translate-x-1">
          <span className="text-foreground text-sm">{transaction.transaction_date}</span>
          <span className="text-foreground text-sm font-medium text-right">
            {formatAmount(transaction.amount, transaction.trade_type)}
          </span>
        </div>

        {/* Row 3: Broker + Upload Date */}
        <div className="flex justify-between items-center mb-2">
          <span className="text-muted text-xs">{transaction.broker}</span>
          <div className="flex items-center gap-2">
            <span className="text-muted text-xs">{transaction.exchange}</span>
            {/* Expand/Collapse indicator */}
            <svg
              className={`w-4 h-4 text-muted transition-transform duration-300 ease-in-out ${
                isExpanded ? 'rotate-180' : ''
              }`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19 9l-7 7-7-7"
              />
            </svg>
          </div>
        </div>

        {/* Action Icons - Always visible */}
        <div className="flex justify-end gap-2 mt-3" data-action>
          <button
            onClick={(e) => {
              e.stopPropagation()
              onEdit(transaction)
            }}
            className="p-1 text-foreground hover:text-primary transition-all duration-200 hover:scale-110"
            title="Edit transaction"
          >
            <EditIcon className="w-4 h-4" />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation()
              onDelete(transaction.id)
            }}
            className="p-1 text-foreground hover:text-chart-1 transition-all duration-200 hover:scale-110"
            title="Delete transaction"
          >
            <DeleteIcon className="w-4 h-4" />
          </button>
        </div>

        {/* Expanded Section with smooth animation */}
        <div
          className={`overflow-hidden transition-all duration-300 ease-in-out ${
            isExpanded ? 'max-h-96 opacity-100' : 'max-h-0 opacity-0'
          }`}
        >
          <div className="mt-4 pt-4 border-t border-border space-y-3">
            {/* Price */}
            <div
              className={`flex justify-between transform transition-all duration-300 hover:translate-x-1 ${
                isExpanded ? 'translate-y-0 opacity-100 delay-75' : 'translate-y-2 opacity-0'
              }`}
            >
              <span className="text-muted text-sm">Price:</span>
              <span className="text-foreground text-sm">${transaction.price.toFixed(2)}</span>
            </div>

            {/* Quantity */}
            <div
              className={`flex justify-between transform transition-all duration-300 hover:translate-x-1 ${
                isExpanded ? 'translate-y-0 opacity-100 delay-100' : 'translate-y-2 opacity-0'
              }`}
            >
              <span className="text-muted text-sm">Quantity:</span>
              <span className="text-foreground text-sm">{transaction.quantity}</span>
            </div>

            {/* Currency */}
            <div
              className={`flex justify-between transform transition-all duration-300 hover:translate-x-1 ${
                isExpanded ? 'translate-y-0 opacity-100 delay-150' : 'translate-y-2 opacity-0'
              }`}
            >
              <span className="text-muted text-sm">Currency:</span>
              <span className="text-foreground text-sm">{transaction.currency}</span>
            </div>

            {/* Notes - Editable */}
            <div
              className={`space-y-2 transform transition-all duration-300 hover:translate-x-1 ${
                isExpanded ? 'translate-y-0 opacity-100 delay-200' : 'translate-y-2 opacity-0'
              }`}
            >
              <span className="text-muted text-sm block">Notes:</span>
              {isEditingNotes ? (
                <Input
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  onBlur={handleNotesSubmit}
                  onKeyDown={handleNotesKeyDown}
                  placeholder="Add notes..."
                  className="bg-input border-border text-foreground text-sm transition-all duration-200 focus:border-primary focus:ring-1 focus:ring-primary/20"
                  autoFocus
                  data-action
                />
              ) : (
                <div
                  onClick={(e) => {
                    e.stopPropagation()
                    setIsEditingNotes(true)
                  }}
                  className="text-foreground text-sm min-h-[32px] p-2 rounded border border-transparent hover:border-border cursor-text transition-all duration-200 hover:bg-input/50"
                  data-action
                >
                  {transaction.user_notes || 'Click to add notes...'}
                </div>
              )}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
