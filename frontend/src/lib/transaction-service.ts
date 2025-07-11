import { apiClient } from './api-client'
import { API_ENDPOINTS } from '@/constants/api'
import type { ExtractResponse } from '@/types'

export class TransactionService {
  /**
   * Extract transactions from a single image file
   * @param file - The image file to process
   * @returns Promise with extraction results
   */
  static async extractTransactions(file: File): Promise<ExtractResponse> {
    const formData = new FormData()
    formData.append('file', file)

    try {
      const response = await apiClient.post<ExtractResponse>(
        API_ENDPOINTS.TRANSACTIONS.EXTRACT,
        formData,
        {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
          timeout: 60000, // 60 seconds for AI processing
        }
      )

      return response.data
    } catch (error: unknown) {
      console.error('Transaction extraction failed:', error)

      // Return a proper error response format
      const errorMessage =
        error instanceof Error
          ? error.message
          : (error as { response?: { data?: { message?: string } } })?.response?.data?.message ||
            'Failed to extract transactions'

      return {
        success: false,
        message: errorMessage,
      }
    }
  }

  /**
   * Process multiple files in parallel
   * @param files - Array of files to process
   * @param onProgress - Callback for progress updates
   * @returns Promise with array of results
   */
  static async extractTransactionsParallel(
    files: File[],
    onProgress?: (fileIndex: number, result: ExtractResponse, error?: string) => void
  ): Promise<ExtractResponse[]> {
    const promises = files.map(async (file, index) => {
      try {
        const result = await this.extractTransactions(file)
        onProgress?.(index, result)
        return result
      } catch (error: unknown) {
        const errorResult: ExtractResponse = {
          success: false,
          message: `Failed to process ${file.name}: ${error instanceof Error ? error.message : 'Unknown error'}`,
        }
        onProgress?.(index, errorResult, error instanceof Error ? error.message : 'Unknown error')
        return errorResult
      }
    })

    return Promise.all(promises)
  }
}
