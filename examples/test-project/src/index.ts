/**
 * A sample TypeScript class for testing LSP functionality
 */
export class Calculator {
  private history: number[] = [];

  /**
   * Adds two numbers and stores the result in history
   * @param a First number
   * @param b Second number
   * @returns The sum of a and b
   */
  add(a: number, b: number): number {
    const result = a + b;
    this.history.push(result);
    return result;
  }

  /**
   * Multiplies two numbers
   * @param a First number
   * @param b Second number
   * @returns The product of a and b
   */
  multiply(a: number, b: number): number {
    const result = a * b;
    this.history.push(result);
    return result;
  }

  /**
   * Gets the calculation history
   * @returns Array of previous results
   */
  getHistory(): number[] {
    return [...this.history];
  }

  /**
   * Clears the calculation history
   */
  clearHistory(): void {
    this.history = [];
  }
}

export interface MathOperation {
  operator: string;
  operands: number[];
  result: number;
}

export type CalculatorMode = 'basic' | 'advanced' | 'scientific';

// Example usage
const calc = new Calculator();
const sum = calc.add(5, 3);
const product = calc.multiply(4, 2);

console.log('Sum:', sum);
console.log('Product:', product);
console.log('History:', calc.getHistory());