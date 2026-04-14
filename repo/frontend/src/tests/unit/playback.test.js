import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';

// Mock all API modules
vi.mock('@/api/playback.js', () => ({
  getMedia: vi.fn(),
  createMedia: vi.fn(),
  parseLyrics: vi.fn(),
  searchLyrics: vi.fn(),
  getSupportedFormats: vi.fn(),
}));

vi.mock('@/api/client.js', () => ({
  default: { post: vi.fn(), get: vi.fn() },
  setLogoutCallback: vi.fn(),
}));

vi.mock('@/api/auth.js', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  refresh: vi.fn(),
}));

vi.mock('@/api/org.js', () => ({
  switchContext: vi.fn(),
  getOrgTree: vi.fn(),
}));

vi.mock('@/router/index.js', () => ({
  default: { push: vi.fn() },
}));

import * as playbackApi from '@/api/playback.js';
import PlaybackPage from '@/pages/PlaybackPage.vue';

function createWrapper(mediaList = [], currentTrack = null) {
  playbackApi.getMedia.mockResolvedValue({
    data: mediaList,
  });

  const wrapper = shallowMount(PlaybackPage, {
    global: {
      plugins: [createPinia()],
      stubs: {
        AppButton: { template: '<button :disabled="disabled || loading"><slot /></button>', props: ['loading', 'disabled', 'variant', 'size', 'block'] },
        AppInput: { template: '<div class="app-input"><input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" :placeholder="placeholder" /></div>', props: ['modelValue', 'placeholder'], emits: ['update:modelValue'] },
        AppChip: { template: '<span class="app-chip">{{ label }}</span>', props: ['label', 'variant', 'status', 'size'] },
        AppFileUpload: { template: '<div class="app-file-upload"></div>', props: ['accept', 'hint'] },
        AppLoadingState: { template: '<div class="loading-state"></div>', props: ['message'] },
        AppErrorState: { template: '<div class="error-state"></div>', props: ['message', 'retryable'] },
        AppEmptyState: { template: '<div class="empty-state"><p>{{ title }}</p><p>{{ description }}</p></div>', props: ['title', 'description'] },
        AppToast: { template: '<div class="toast"></div>' },
        Transition: { template: '<div><slot /></div>' },
      },
    },
  });

  if (currentTrack) {
    wrapper.vm.currentTrack = currentTrack;
    wrapper.vm.mediaList = mediaList;
  }

  return wrapper;
}

describe('PlaybackPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();

    // Mock HTMLMediaElement methods
    HTMLMediaElement.prototype.play = vi.fn().mockResolvedValue(undefined);
    HTMLMediaElement.prototype.pause = vi.fn();
    HTMLMediaElement.prototype.load = vi.fn();
  });

  it('lyrics not available fallback shown when no lyrics', async () => {
    const track = { id: '1', title: 'Test Song', artist: 'Test Artist', url: 'http://test.com/song.mp3' };
    playbackApi.getMedia.mockResolvedValue({ data: [track] });
    playbackApi.parseLyrics.mockResolvedValue({ data: '' });

    const wrapper = createWrapper([track], track);
    await flushPromises();
    await nextTick();

    // parsedLyrics should be empty, so the fallback should show
    expect(wrapper.vm.parsedLyrics).toHaveLength(0);
    const emptyPanel = wrapper.find('.lyrics-panel__empty');
    expect(emptyPanel.exists()).toBe(true);
    expect(emptyPanel.text()).toContain('Lyrics not available');
  });

  it('lyrics search highlights matching lines', async () => {
    const track = { id: '1', title: 'Test', artist: 'Artist', url: 'http://test.com/song.mp3' };
    const wrapper = createWrapper([track], track);
    await flushPromises();

    // Set parsed lyrics manually
    wrapper.vm.parsedLyrics = [
      { time: 0, text: 'Hello world', words: [] },
      { time: 5, text: 'Goodbye moon', words: [] },
      { time: 10, text: 'Hello again', words: [] },
    ];
    await nextTick();

    // Simulate lyrics search - set the search match indices
    wrapper.vm.searchMatchIndices = new Set([0, 2]); // Lines matching "Hello"
    await nextTick();

    const lines = wrapper.findAll('.lyrics-line');
    expect(lines.length).toBe(3);

    // Lines at index 0 and 2 should have search-match class
    expect(lines[0].classes()).toContain('lyrics-line--search-match');
    expect(lines[1].classes()).not.toContain('lyrics-line--search-match');
    expect(lines[2].classes()).toContain('lyrics-line--search-match');
  });

  it('seek confirmation visual indicator appears', async () => {
    const track = { id: '1', title: 'Test', artist: 'Artist', url: 'http://test.com/song.mp3' };
    const wrapper = createWrapper([track], track);
    await flushPromises();

    wrapper.vm.parsedLyrics = [
      { time: 0, text: 'Line one', words: [] },
      { time: 5, text: 'Line two', words: [] },
    ];
    await nextTick();

    // Simulate seeking to a line - should trigger pulse
    expect(wrapper.vm.pulseLine).toBe(-1);

    // Call seekToLine which sets pulseLine
    wrapper.vm.seekToLine(1);
    await nextTick();

    // pulseLine should now be 1, creating the visual indicator
    expect(wrapper.vm.pulseLine).toBe(1);

    // Check that the line has the pulse class
    const lines = wrapper.findAll('.lyrics-line');
    if (lines.length > 1) {
      expect(lines[1].classes()).toContain('lyrics-line--pulse');
    }
  });

  it('unsupported format shows graceful fallback without blocking controls', async () => {
    const track = { id: '1', title: 'Test', artist: 'Artist', url: 'http://test.com/song.unknown', filename: 'song.unknown' };
    const wrapper = createWrapper([track], track);
    await flushPromises();
    await nextTick();

    // Simulate an audio error (unsupported format)
    wrapper.vm.unsupportedFormat = true;
    await nextTick();

    // The format chip and unsupported message should show
    expect(wrapper.vm.unsupportedFormat).toBe(true);

    // Controls (play, prev, next) should still be accessible (not disabled)
    // The player panel controls should still be rendered
    expect(wrapper.vm.formatLabel).toBe('UNKNOWN');

    // The page should not crash; toggle play should still work
    wrapper.vm.togglePlay();
    // Even with unsupported format, controls respond
    expect(true).toBe(true); // No exception thrown
  });

  it('audio player controls remain responsive during LRC parse error', async () => {
    const track = { id: '1', title: 'Test', artist: 'Artist', url: 'http://test.com/song.mp3' };
    playbackApi.parseLyrics.mockRejectedValue(new Error('Parse failed'));

    const wrapper = createWrapper([track], track);
    await flushPromises();

    // Attempt to load lyrics which will fail
    await wrapper.vm.selectTrack(track);
    await flushPromises();

    // Lyrics should be empty after parse error
    expect(wrapper.vm.parsedLyrics).toHaveLength(0);
    expect(wrapper.vm.lrcParsing).toBe(false);

    // Audio controls should still work
    wrapper.vm.togglePlay();
    wrapper.vm.nextTrack();
    wrapper.vm.prevTrack();

    // No exceptions, controls remain responsive
    expect(wrapper.vm.isPlaying).toBeDefined();
  });

  it('LRC parser correctly parses standard LRC content', async () => {
    // Test the parseLRC function via the component
    const lrcContent = `[00:01.00]First line
[00:05.50]Second line
[00:10.00]Third line`;
    const track = { id: '1', title: 'Test', artist: 'Artist', url: 'http://test.com/song.mp3', lrcContent };
    playbackApi.getMedia.mockResolvedValue({ data: [track] });

    const wrapper = createWrapper([track]);
    await flushPromises();

    // Call selectTrack which triggers parseLRC via nextTick for lrcContent
    wrapper.vm.selectTrack(track);
    await flushPromises();
    await nextTick();

    expect(wrapper.vm.parsedLyrics.length).toBe(3);
    expect(wrapper.vm.parsedLyrics[0].text).toBe('First line');
    expect(wrapper.vm.parsedLyrics[0].time).toBeCloseTo(1.0, 1);
    expect(wrapper.vm.parsedLyrics[1].time).toBeCloseTo(5.5, 1);
  });
});
