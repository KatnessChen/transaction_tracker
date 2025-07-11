import { setValidationError } from '@/store/fileProcessingSlice'
import type { AppDispatch } from '@/store'

export function validateTransactionDate(value: string, errorKey: string, dispatch: AppDispatch) {
  const inputDate = new Date(value)
  const today = new Date()
  const thirtyYearsAgo = new Date()
  thirtyYearsAgo.setFullYear(today.getFullYear() - 30)

  if (isNaN(inputDate.getTime())) {
    dispatch(setValidationError({ errorKey, message: 'Please enter a valid date.' }))
    return false
  }
  if (inputDate > today) {
    dispatch(
      setValidationError({
        errorKey,
        message: 'Transaction date cannot be in the future.',
      })
    )
    return false
  }
  if (inputDate < thirtyYearsAgo) {
    dispatch(
      setValidationError({
        errorKey,
        message: 'Transaction date must be within the last 30 years.',
      })
    )
    return false
  }
  return true
}

export function validateTransactionQuantity(
  tradeType: string,
  value: number,
  errorKey: string,
  dispatch: AppDispatch
) {
  if (tradeType === 'Dividends') return true
  if (isNaN(value) || value === 0) {
    dispatch(
      setValidationError({
        errorKey,
        message: 'Quantity cannot be zero.',
      })
    )
    return false
  }
  return true
}

export function validateTransactionPrice(
  tradeType: string,
  value: number,
  errorKey: string,
  dispatch: AppDispatch
) {
  if (tradeType === 'Dividends') return true
  if (isNaN(value) || value <= 0) {
    dispatch(
      setValidationError({
        errorKey,
        message: 'Price must > 0.',
      })
    )
    return false
  }
  return true
}
