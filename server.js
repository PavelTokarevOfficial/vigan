require('dotenv').config();

const express = require('express');
const path = require('path');
const fs = require('fs/promises');
const { createWriteStream } = require('fs');
const { finished } = require('stream/promises');
const { Readable } = require('stream');
const { execFile } = require('child_process');
const { promisify } = require('util');
const { randomUUID } = require('crypto');
const { chromium } = require('playwright');

const app = express();
const PORT = process.env.PORT || 3000;
const DOWNLOADS_DIR = path.join(__dirname, 'downloads');
const WHISPER_MODEL_PATH = path.join(__dirname, 'models', 'ggml-tiny.bin');
const FFMPEG_PATH = process.env.FFMPEG_PATH || (process.arch === 'arm64'
  ? '/opt/homebrew/opt/ffmpeg-full/bin/ffmpeg'
  : '/usr/local/opt/ffmpeg-full/bin/ffmpeg');
const VIDEO_EXTENSIONS = new Set(['.mp4', '.webm', '.mov', '.mkv']);
const transcriptionJobs = new Set();
const execFileAsync = promisify(execFile);
const { TWITCH_CLIENT_ID, TWITCH_CLIENT_SECRET } = process.env;

let tokenCache = {
  accessToken: null,
  expiresAt: 0,
};

app.use(express.static(path.join(__dirname, 'public')));
app.use('/downloads', express.static(DOWNLOADS_DIR));

function isVideoFile(fileName) {
  return path.basename(fileName) === fileName && VIDEO_EXTENSIONS.has(path.extname(fileName).toLowerCase());
}

async function listDownloadedVideos() {
  await fs.mkdir(DOWNLOADS_DIR, { recursive: true });
  const entries = await fs.readdir(DOWNLOADS_DIR, { withFileTypes: true });
  const videos = await Promise.all(entries
    .filter((entry) => entry.isFile() && isVideoFile(entry.name))
    .map(async (entry) => {
      const info = await fs.stat(path.join(DOWNLOADS_DIR, entry.name));
      const subtitleName = `${path.parse(entry.name).name}.srt`;
      const srtName = `${path.parse(entry.name).name}.srt`;
      try {
        await fs.access(path.join(DOWNLOADS_DIR, srtName));
        return {
          name: entry.name,
          size: info.size,
          modifiedAt: info.mtime.toISOString(),
          subtitleName,
          subtitleState: 'ready',
          srtName,
        };
      } catch {
        return {
          name: entry.name,
          size: info.size,
          modifiedAt: info.mtime.toISOString(),
          subtitleName: null,
          subtitleState: null,
          srtName: null,
        };
      }
    }));
  return videos.sort((left, right) => new Date(right.modifiedAt) - new Date(left.modifiedAt));
}

function downloadFileName(clipId) {
  return `twitch-clip-${clipId.replace(/[^a-zA-Z0-9_-]/g, '_')}.mp4`;
}

async function unusedDownloadPath(fileName) {
  const parsed = path.parse(fileName);
  for (let suffix = 0; ; suffix += 1) {
    const candidate = path.join(DOWNLOADS_DIR, `${parsed.name}${suffix ? `-${suffix}` : ''}${parsed.ext}`);
    try {
      await fs.access(candidate);
    } catch {
      return candidate;
    }
  }
}

async function getClipVideoSource(clipUrl) {
  const browser = await chromium.launch({ headless: true });
  try {
    const page = await browser.newPage();
    await page.goto(clipUrl, { waitUntil: 'domcontentloaded', timeout: 30_000 });
    // Twitch renders the player asynchronously; this reads the media source from its video element.
    const video = page.locator('video').first();
    await video.waitFor({ state: 'attached', timeout: 20_000 });
    await page.waitForFunction(
      () => {
        const element = document.querySelector('video');
        return Boolean(element?.currentSrc || element?.src);
      },
      { timeout: 20_000 },
    );
    const source = await video.evaluate((element) => element.currentSrc || element.src);
    if (!source) throw new Error('The Twitch player did not expose a video source.');
    return { source, userAgent: await page.evaluate(() => navigator.userAgent) };
  } finally {
    await browser.close();
  }
}

async function saveClipVideo({ clipId, clipUrl }) {
  await fs.mkdir(DOWNLOADS_DIR, { recursive: true });
  const { source, userAgent } = await getClipVideoSource(clipUrl);
  if (!/^https?:\/\//i.test(source)) {
    throw new Error('The Twitch player returned an unsupported video source.');
  }

  const response = await fetch(source, {
    headers: { 'User-Agent': userAgent, Referer: clipUrl },
    signal: AbortSignal.timeout(120_000),
  });
  if (!response.ok || !response.body) {
    throw new Error(`Could not download the video file (${response.status}).`);
  }

  const destination = await unusedDownloadPath(downloadFileName(clipId));
  const stream = createWriteStream(destination, { flags: 'wx' });
  try {
    await finished(Readable.fromWeb(response.body).pipe(stream));
  } catch (error) {
    await fs.rm(destination, { force: true });
    throw error;
  }
  return path.basename(destination);
}

async function getAppAccessToken() {
  // Refresh slightly early so a token does not expire during a Helix request.
  if (tokenCache.accessToken && Date.now() < tokenCache.expiresAt) {
    return tokenCache.accessToken;
  }

  if (!TWITCH_CLIENT_ID || !TWITCH_CLIENT_SECRET) {
    throw new Error('Twitch API credentials are not configured. Fill in the .env file.');
  }

  const body = new URLSearchParams({
    client_id: TWITCH_CLIENT_ID,
    client_secret: TWITCH_CLIENT_SECRET,
    grant_type: 'client_credentials',
  });

  const response = await fetch('https://id.twitch.tv/oauth2/token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body,
  });

  if (!response.ok) {
    throw new Error('Could not get an access token from Twitch. Check your Client ID and Client Secret.');
  }

  const data = await response.json();
  tokenCache = {
    accessToken: data.access_token,
    expiresAt: Date.now() + Math.max(0, data.expires_in - 60) * 1000,
  };

  return tokenCache.accessToken;
}

async function twitchGet(endpoint, searchParams) {
  const accessToken = await getAppAccessToken();
  const url = new URL(`https://api.twitch.tv/helix/${endpoint}`);
  url.search = new URLSearchParams(searchParams).toString();

  const response = await fetch(url, {
    headers: {
      'Client-Id': TWITCH_CLIENT_ID,
      Authorization: `Bearer ${accessToken}`,
    },
  });

  if (!response.ok) {
    throw new Error(`Twitch API request failed (${response.status}). Please try again later.`);
  }

  return response.json();
}

app.get('/api/clips', async (req, res) => {
  const username = String(req.query.username || '').trim();
  const after = String(req.query.after || '').trim();
  const startedAt = String(req.query.started_at || '').trim();
  const endedAt = String(req.query.ended_at || '').trim();
  const minViewsRaw = String(req.query.min_views || '').trim();

  if (!username) {
    return res.status(400).json({ error: 'Enter a Twitch username.' });
  }

  // Twitch logins are 4-25 characters and use letters, numbers, and underscores.
  if (!/^[a-zA-Z0-9_]{4,25}$/.test(username)) {
    return res.status(400).json({ error: 'Username must contain 4-25 letters, numbers, or underscores.' });
  }

  if ((startedAt && !endedAt) || (!startedAt && endedAt)) {
    return res.status(400).json({ error: 'Specify both the start and end of the period.' });
  }

  if (startedAt && (Number.isNaN(Date.parse(startedAt)) || Number.isNaN(Date.parse(endedAt)))) {
    return res.status(400).json({ error: 'Invalid date range.' });
  }

  if (startedAt && Date.parse(startedAt) > Date.parse(endedAt)) {
    return res.status(400).json({ error: 'The start of the period must be before its end.' });
  }

  const minViews = minViewsRaw === '' ? 0 : Number(minViewsRaw);
  if (!Number.isInteger(minViews) || minViews < 0) {
    return res.status(400).json({ error: 'Minimum views must be a non-negative integer.' });
  }

  try {
    const users = await twitchGet('users', { login: username });
    const broadcaster = users.data?.[0];

    if (!broadcaster) {
      return res.status(404).json({ error: `Twitch user "${username}" was not found.` });
    }

    const query = { broadcaster_id: broadcaster.id, first: '100' };
    if (after) query.after = after;
    if (startedAt) {
      query.started_at = new Date(startedAt).toISOString();
      query.ended_at = new Date(endedAt).toISOString();
    }

    const clipsResult = await twitchGet('clips', query);
    // Get Clips is already sorted by views descending. Twitch does not expose a
    // min-views parameter, so the threshold is applied to this returned page.
    const clips = (clipsResult.data || []).filter((clip) => (clip.view_count || 0) >= minViews);
    return res.json({
      broadcaster: {
        id: broadcaster.id,
        login: broadcaster.login,
        displayName: broadcaster.display_name,
      },
      clips,
      pagination: {
        cursor: clipsResult.pagination?.cursor || null,
      },
    });
  } catch (error) {
    console.error('Clips request error:', error.message);
    return res.status(500).json({ error: error.message || 'Unexpected server error.' });
  }
});

app.post('/api/clips/:clipId/download', express.json(), async (req, res) => {
  const clipId = String(req.params.clipId || '').trim();
  const clipUrl = String(req.body?.url || '').trim();

  if (!/^[a-zA-Z0-9_-]+$/.test(clipId) || !/^https:\/\/(?:www\.)?twitch\.tv\//i.test(clipUrl)) {
    return res.status(400).json({ error: 'Invalid Twitch clip.' });
  }

  try {
    const fileName = await saveClipVideo({ clipId, clipUrl });
    return res.json({ message: `Клип сохранён в downloads/${fileName}.`, fileName });
  } catch (error) {
    console.error('Clip download error:', error.message);
    return res.status(502).json({ error: error.message || 'Could not download the clip.' });
  }
});

app.get('/api/downloads', async (_req, res) => {
  try {
    return res.json({ videos: await listDownloadedVideos() });
  } catch (error) {
    console.error('Downloads list error:', error.message);
    return res.status(500).json({ error: 'Не удалось прочитать папку downloads.' });
  }
});

app.post('/api/downloads/:fileName/subtitles', async (req, res) => {
  const fileName = String(req.params.fileName || '');
  if (!isVideoFile(fileName)) {
    return res.status(400).json({ error: 'Недопустимое имя видеофайла.' });
  }

  const videoPath = path.join(DOWNLOADS_DIR, fileName);
  const videoBaseName = path.parse(fileName).name;
  const subtitleName = `${videoBaseName}.srt`;
  const subtitlePath = path.join(DOWNLOADS_DIR, subtitleName);
  const temporaryAudioPath = path.join(DOWNLOADS_DIR, `.${videoBaseName}-${randomUUID()}.wav`);

  try {
    await fs.access(videoPath);
    await fs.access(WHISPER_MODEL_PATH);
    if (transcriptionJobs.has(fileName)) {
      return res.status(409).json({ error: 'Для этого видео уже создаются субтитры.' });
    }

    transcriptionJobs.add(fileName);
    const outputBasePath = path.join(DOWNLOADS_DIR, videoBaseName);
    try {
      // Convert the downloaded video to mono WAV, then whisper.cpp writes the transcript as .txt beside it.
      await execFileAsync(FFMPEG_PATH, ['-y', '-i', videoPath, '-ar', '16000', '-ac', '1', temporaryAudioPath], {
        timeout: 120_000,
        maxBuffer: 1024 * 1024,
      });
      await execFileAsync('whisper-cli', [
        '-m', WHISPER_MODEL_PATH,
        '-l', 'auto',
        '-osrt',
        '-of', outputBasePath,
        '-f', temporaryAudioPath,
      ], { timeout: 15 * 60_000, maxBuffer: 1024 * 1024 });

      const videoWithSubtitlesPath = await unusedDownloadPath(`width-sub-${videoBaseName}.mp4`);
      // SRT carries the cue timings; FFmpeg burns each cue into the center of the new video.
      await execFileAsync(FFMPEG_PATH, [
        '-y', '-i', videoPath,
        '-vf', `subtitles=filename='${subtitlePath}':force_style='Alignment=5,Fontsize=22,PrimaryColour=&H00FFFFFF,OutlineColour=&H00000000,BorderStyle=1,Outline=2'`,
        '-c:v', 'libx264', '-preset', 'veryfast', '-crf', '20',
        '-c:a', 'aac', '-movflags', '+faststart',
        videoWithSubtitlesPath,
      ], { timeout: 15 * 60_000, maxBuffer: 1024 * 1024 });

      // Older MVP runs could have created these auxiliary files; SRT is now the single subtitle source.
      await Promise.all([
        fs.rm(path.join(DOWNLOADS_DIR, `${videoBaseName}.txt`), { force: true }),
        fs.rm(path.join(DOWNLOADS_DIR, `${videoBaseName}.vtt`), { force: true }),
      ]);
      return res.status(201).json({
        message: `Готово: SRT и видео с субтитрами сохранены в downloads/${path.basename(videoWithSubtitlesPath)}.`,
        subtitleName,
        videoName: path.basename(videoWithSubtitlesPath),
        exists: false,
      });
    } finally {
      transcriptionJobs.delete(fileName);
      await fs.rm(temporaryAudioPath, { force: true });
    }
  } catch {
    return res.status(500).json({ error: 'Не удалось создать субтитры. Проверьте, что ffmpeg и whisper-cli установлены.' });
  }
});

app.listen(PORT, () => {
  console.log(`Finde Clip is running at http://localhost:${PORT}`);
});
