import { Calculator } from './index';

/**
 * Utility functions for mathematical operations
 */

/**
 * Calculates the factorial of a number
 * @param n The number to calculate factorial for
 * @returns The factorial of n
 */
export function factorial(n: number): number {
  if (n <= 1) return 1;
  return n * factorial(n - 1);
}

/**
 * Checks if a number is prime
 * @param num The number to check
 * @returns True if the number is prime, false otherwise
 */
export function isPrime(num: number): boolean {
  if (num <= 1) return false;
  if (num <= 3) return true;
  if (num % 2 === 0 || num % 3 === 0) return false;
  
  for (let i = 5; i * i <= num; i += 6) {
    if (num % i === 0 || num % (i + 2) === 0) {
      return false;
    }
  }
  return true;
}

/**
 * Creates a calculator instance with predefined operations
 * @returns A new Calculator instance
 */
export function createCalculator(): Calculator {
  return new Calculator();
}