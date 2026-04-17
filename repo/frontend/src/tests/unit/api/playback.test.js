import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import * as api from '@/api/playback.js';

describe('api/playback.js — full coverage', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getMedia', async () => { ok([]); await api.getMedia({ search: 'test' }); expect(captured.url).toBe('/media'); });
  it('createMedia', async () => { ok({ id: 1 }); await api.createMedia({ title: 'Song' }); expect(captured.method).toBe('post'); });
  it('getMediaById', async () => { ok({ id: 1 }); await api.getMediaById(7); expect(captured.url).toBe('/media/7'); });
  it('updateMedia', async () => { ok(); await api.updateMedia(3, { title: 'X' }); expect(captured.method).toBe('put'); });
  it('deleteMedia', async () => { ok(); await api.deleteMedia(3); expect(captured.method).toBe('delete'); });
  it('streamAudio fetches blob', async () => {
    ok(new Blob(['audio'], { type: 'audio/mpeg' }));
    await api.streamAudio(5);
    expect(captured.url).toBe('/media/5/stream');
    expect(captured.responseType).toBe('blob');
  });
  it('getCoverArt fetches blob', async () => {
    ok(new Blob(['img'], { type: 'image/jpeg' }));
    await api.getCoverArt(5);
    expect(captured.url).toBe('/media/5/cover');
    expect(captured.responseType).toBe('blob');
  });
  it('parseLyrics', async () => { ok({}); await api.parseLyrics(5); expect(captured.url).toBe('/media/5/lyrics/parse'); expect(captured.method).toBe('post'); });
  it('searchLyrics', async () => { ok({ matches: [] }); await api.searchLyrics(5, 'hello'); expect(captured.url).toBe('/media/5/lyrics/search'); });
  it('getSupportedFormats', async () => { ok({ formats: [] }); await api.getSupportedFormats(); expect(captured.url).toBe('/media/formats/supported'); });
});
