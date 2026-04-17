import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import { getMedia, getMediaById, createMedia, updateMedia, deleteMedia } from '@/api/playback.js';

describe('api/playback.js', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getMedia sends GET /media with params', async () => { ok([]); await getMedia({ search: 'test' }); expect(captured.url).toBe('/media'); expect(captured.params.search).toBe('test'); });
  it('getMediaById sends GET /media/:id', async () => { ok({ id: 1 }); await getMediaById(7); expect(captured.url).toBe('/media/7'); });
  it('createMedia sends POST /media', async () => { ok({ id: 1 }); await createMedia({ title: 'Song' }); expect(captured.method).toBe('post'); expect(captured.url).toBe('/media'); });
  it('updateMedia sends PUT /media/:id', async () => { ok(); await updateMedia(3, { title: 'X' }); expect(captured.url).toBe('/media/3'); expect(captured.method).toBe('put'); });
  it('deleteMedia sends DELETE /media/:id', async () => { ok(); await deleteMedia(3); expect(captured.url).toBe('/media/3'); expect(captured.method).toBe('delete'); });
});
