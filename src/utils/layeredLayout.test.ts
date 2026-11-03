import { layeredLayout } from './layeredLayout';

describe('layeredLayout', () => {
  it('returns no positions for an empty graph', () => {
    expect(layeredLayout([], []).size).toBe(0);
  });

  it('places callers left of callees', () => {
    const pos = layeredLayout(
      ['gateway', 'api', 'db'],
      [
        { source: 'gateway', target: 'api' },
        { source: 'api', target: 'db' },
      ]
    );
    expect(pos.get('gateway')!.x).toBeLessThan(pos.get('api')!.x);
    expect(pos.get('api')!.x).toBeLessThan(pos.get('db')!.x);
  });

  it('gives every node a finite, unique position', () => {
    const ids = ['a', 'b', 'c', 'd', 'e', 'lonely'];
    const pos = layeredLayout(ids, [
      { source: 'a', target: 'b' },
      { source: 'a', target: 'c' },
      { source: 'b', target: 'd' },
      { source: 'c', target: 'd' },
      { source: 'd', target: 'e' },
    ]);
    expect(pos.size).toBe(ids.length);
    const keys = new Set<string>();
    for (const id of ids) {
      const p = pos.get(id)!;
      expect(Number.isFinite(p.x)).toBe(true);
      expect(Number.isFinite(p.y)).toBe(true);
      keys.add(`${p.x},${p.y}`);
    }
    expect(keys.size).toBe(ids.length);
  });

  it('terminates on cycles, self-loops and dangling edges', () => {
    const pos = layeredLayout(
      ['a', 'b', 'c'],
      [
        { source: 'a', target: 'b' },
        { source: 'b', target: 'c' },
        { source: 'c', target: 'a' },
        { source: 'b', target: 'b' },
        { source: 'b', target: 'missing' },
      ]
    );
    expect(pos.size).toBe(3);
    expect(pos.get('a')!.x).toBeLessThan(pos.get('b')!.x);
    expect(pos.get('b')!.x).toBeLessThan(pos.get('c')!.x);
  });

  it('centres the layout on the origin', () => {
    const pos = layeredLayout(['a', 'b'], [{ source: 'a', target: 'b' }]);
    expect(pos.get('a')!.x + pos.get('b')!.x).toBe(0);
    expect(pos.get('a')!.y).toBe(0);
  });

  it('pulls a source next to its callee rather than layer zero', () => {
    const pos = layeredLayout(
      ['a', 'b', 'c', 'cron'],
      [
        { source: 'a', target: 'b' },
        { source: 'b', target: 'c' },
        { source: 'cron', target: 'c' },
      ]
    );
    expect(pos.get('cron')!.x).toBe(pos.get('b')!.x);
  });

  it('orders a layer to avoid crossings', () => {
    const pos = layeredLayout(
      ['a1', 'a2', 'b2', 'b1'],
      [
        { source: 'a1', target: 'b1' },
        { source: 'a2', target: 'b2' },
      ]
    );
    expect(Math.sign(pos.get('a1')!.y - pos.get('a2')!.y)).toBe(Math.sign(pos.get('b1')!.y - pos.get('b2')!.y));
  });
});
