import { describe, expect, it } from 'vitest';
import { unwrap } from './apiTypes';

describe('unwrap', () => {
  it('returns the data payload from the API envelope', () => {
    expect(unwrap({ data: { id: 7, label: 'inventory' } })).toEqual({ id: 7, label: 'inventory' });
  });
});