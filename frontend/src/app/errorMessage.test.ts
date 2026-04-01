import { describe, expect, it } from 'vitest';

import { extractApiErrorMessage } from './errorMessage';

describe('extractApiErrorMessage', () => {
  it('returns envelope error messages from RTK Query responses', () => {
    expect(
      extractApiErrorMessage(
        {
          status: 409,
          data: { error: 'insufficient available stock for sku SKU-001' },
        },
        'fallback',
      ),
    ).toBe('insufficient available stock for sku SKU-001');
  });

  it('falls back when the error shape is unknown', () => {
    expect(extractApiErrorMessage(null, 'fallback')).toBe('fallback');
  });
});