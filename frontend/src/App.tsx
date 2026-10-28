import React, { useState, useEffect } from 'react';
import { 
  Link2, 
  Copy, 
  Download, 
  ArrowRight, 
  Loader2, 
  Check, 
  AlertCircle 
} from 'lucide-react';

interface ConvertResponse {
  task_id: string;
  filename: string;
  filesize: string;
  download_url: string;
}

interface NetworkInfo {
  lan_ip: string;
  port: number;
}

export const App: React.FC = () => {
  const [url, setUrl] = useState<string>('');
  const [bitrate, setBitrate] = useState<'192' | '320' | '128'>('192');
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<ConvertResponse | null>(null);
  const [lanIp, setLanIp] = useState<string>('');
  const [toast, setToast] = useState<string | null>(null);

  useEffect(() => {
    // Fetch LAN IP from backend
    fetch('/api/network')
      .then(res => res.json())
      .then((data: NetworkInfo) => {
        if (data.lan_ip) {
          setLanIp(`http://${data.lan_ip}:${data.port || 8000}`);
        }
      })
      .catch(() => {
        // Fallback to window host
        setLanIp(`http://${window.location.hostname}:8000`);
      });
  }, []);

  const showToast = (msg: string) => {
    setToast(msg);
    setTimeout(() => {
      setToast(null);
    }, 2000);
  };

  const handleCopyIp = async () => {
    if (!lanIp) return;
    try {
      await navigator.clipboard.writeText(lanIp);
      showToast('Alamat URL disalin');
    } catch {
      showToast('Gagal menyalin');
    }
  };

  const handlePaste = async () => {
    try {
      const text = await navigator.clipboard.readText();
      if (text) {
        setUrl(text);
        showToast('Tautan ditempel');
      }
    } catch {
      showToast('Izinkan akses clipboard di browser');
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const cleanUrl = url.trim();
    if (!cleanUrl) return;

    setError(null);
    setResult(null);
    setLoading(true);

    try {
      const res = await fetch('/api/convert', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          url: cleanUrl,
          bitrate,
          embed_thumbnail: true,
        }),
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.detail || 'Terjadi kesalahan saat memproses');
      }

      setResult(data);

      // Trigger automatic file download
      const a = document.createElement('a');
      a.href = data.download_url;
      a.download = data.filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);

      showToast('Download dimulai');
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Terjadi kesalahan yang tidak diketahui');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-col font-sans selection:bg-zinc-800 selection:text-white bg-[#09090b] text-zinc-100">
      
      {/* Top Header */}
      <header className="border-b border-zinc-800/80 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-7 h-7 rounded-lg bg-zinc-100 text-zinc-950 flex items-center justify-center font-mono font-bold text-xs tracking-tight shadow-sm">
            MP3
          </div>
          <div>
            <span className="text-sm font-semibold tracking-tight text-zinc-100">Audio Extractor</span>
            <span className="text-xs text-zinc-500 font-mono ml-2 hidden sm:inline">v1.0 (TS)</span>
          </div>
        </div>

        {/* LAN Access Chip */}
        {lanIp && (
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-[#121215] border border-zinc-800 text-xs text-zinc-400 font-mono">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
              <span className="hidden md:inline text-zinc-500 font-sans">HP:</span>
              <span className="text-zinc-200">{lanIp}</span>
            </div>
            <button 
              type="button"
              onClick={handleCopyIp}
              title="Salin alamat URL untuk dibuka di HP"
              className="h-7 px-2.5 rounded-md bg-zinc-800 hover:bg-zinc-700 active:bg-zinc-600 text-xs text-zinc-200 transition-colors flex items-center gap-1"
            >
              <Copy className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Salin</span>
            </button>
          </div>
        )}
      </header>

      {/* Main Container */}
      <main className="flex-1 flex flex-col justify-center px-4 py-12 max-w-2xl mx-auto w-full">
        <div className="mb-8">
          <h1 className="text-2xl font-semibold tracking-tight text-zinc-100">Konversi ke MP3</h1>
          <p className="text-sm text-zinc-400 mt-1">Ekstrak audio dari YouTube &amp; TikTok berkualitas tinggi ke format MP3.</p>
        </div>

        {/* Input Box */}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="rounded-xl border border-zinc-800 bg-[#121215] p-2 focus-within:border-zinc-500 transition-colors shadow-sm">
            <div className="flex items-center gap-2 px-2 py-1">
              <Link2 className="w-4 h-4 text-zinc-500 flex-shrink-0" />
              <input 
                type="url" 
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                required
                placeholder="Masukkan link YouTube atau TikTok..." 
                className="w-full bg-transparent text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none"
              />
              <button 
                type="button" 
                onClick={handlePaste}
                className="text-xs px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 transition-colors flex-shrink-0 font-medium"
              >
                Paste
              </button>
            </div>

            <div className="mt-2 pt-2 border-t border-zinc-800/60 flex flex-wrap items-center justify-between gap-3 px-2">
              {/* Bitrate Selector */}
              <div className="flex items-center gap-1 bg-[#18181b] p-1 rounded-lg border border-zinc-800 text-xs">
                <span className="text-[11px] text-zinc-500 px-1 font-mono uppercase">Bitrate:</span>
                {(['192', '320', '128'] as const).map((b) => (
                  <button
                    key={b}
                    type="button"
                    onClick={() => setBitrate(b)}
                    className={`px-2.5 py-1 rounded text-xs font-mono transition-colors ${
                      bitrate === b
                        ? 'bg-zinc-200 text-zinc-950 font-medium'
                        : 'text-zinc-400 hover:text-zinc-200'
                    }`}
                  >
                    {b}k
                  </button>
                ))}
              </div>

              {/* Submit Button */}
              <button 
                type="submit" 
                disabled={loading}
                className={`h-8 px-4 bg-zinc-100 hover:bg-white active:bg-zinc-200 text-zinc-950 text-xs font-medium rounded-lg transition-colors flex items-center gap-1.5 shadow-sm ${
                  loading ? 'opacity-60 cursor-not-allowed' : ''
                }`}
              >
                {loading ? (
                  <>
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    <span>Mengonversi...</span>
                  </>
                ) : (
                  <>
                    <span>Ekstrak &amp; Unduh</span>
                    <ArrowRight className="w-3.5 h-3.5" />
                  </>
                )}
              </button>
            </div>
          </div>
        </form>

        {/* Processing Indicator */}
        {loading && (
          <div className="mt-6 rounded-xl border border-zinc-800 bg-[#121215] p-4 flex items-center gap-3 animate-fade-in">
            <Loader2 className="w-4 h-4 text-zinc-400 animate-spin flex-shrink-0" />
            <div className="flex-1 min-w-0">
              <p className="text-xs font-medium text-zinc-200">Memproses media...</p>
              <p className="text-[11px] text-zinc-500 mt-0.5 font-mono">Mengunduh stream &amp; encoding MP3</p>
            </div>
          </div>
        )}

        {/* Error Notification */}
        {error && (
          <div className="mt-6 rounded-xl border border-red-900/60 bg-red-950/20 p-4 text-xs text-red-300 flex items-start gap-3">
            <AlertCircle className="w-4 h-4 text-red-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="font-medium text-red-200">Gagal mengonversi</p>
              <p className="mt-1 text-red-400 font-mono text-[11px]">{error}</p>
            </div>
          </div>
        )}

        {/* Success / Download Card */}
        {result && (
          <div className="mt-6 rounded-xl border border-zinc-800 bg-[#121215] p-4 flex items-center justify-between gap-4 animate-fade-in">
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <Check className="w-3.5 h-3.5 text-emerald-400" />
                <span className="text-xs text-zinc-400 font-mono">MP3 Siap</span>
              </div>
              <p className="text-sm font-medium text-zinc-100 truncate mt-1">{result.filename}</p>
              <p className="text-xs text-zinc-500 font-mono mt-0.5">{result.filesize} • {bitrate} kbps</p>
            </div>

            <a 
              href={result.download_url} 
              download={result.filename}
              className="h-9 px-4 rounded-lg bg-zinc-800 hover:bg-zinc-700 active:bg-zinc-600 text-xs font-medium text-white transition-colors flex items-center gap-1.5 flex-shrink-0"
            >
              <Download className="w-3.5 h-3.5" />
              <span>Unduh Ulang</span>
            </a>
          </div>
        )}
      </main>

      {/* Footer */}
      <footer className="border-t border-zinc-800/60 px-6 py-4 text-center text-xs text-zinc-600 font-mono">
        Go Fiber v3 + React TypeScript — Auto-cleaned storage
      </footer>

      {/* Toast */}
      {toast && (
        <div className="fixed bottom-5 right-5 bg-zinc-100 text-zinc-950 font-medium text-xs px-3.5 py-2 rounded-md shadow-lg transition-opacity">
          {toast}
        </div>
      )}

    </div>
  );
};
