import { ExternalLink, Info } from 'lucide-react';

const releaseUrl = `https://github.com/banshee86vr/snorlx/releases/tag/v${__APP_VERSION__}`;

export function AboutCard() {
  return (
    <div className="card">
      <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2">
          <Info className="w-5 h-5 text-gray-500" />
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">About</h2>
        </div>
      </div>
      <div className="p-6">
        <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <div>
            <p className="font-medium text-gray-900 dark:text-gray-100">Version</p>
            <p className="text-sm text-gray-500 dark:text-gray-400">Frontend build</p>
          </div>
          <span className="text-sm text-gray-600 dark:text-gray-300">{__APP_VERSION__}</span>
        </div>
        <a
          href={releaseUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 mt-4 text-sm text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100"
        >
          GitHub release
          <ExternalLink className="w-4 h-4" />
        </a>
      </div>
    </div>
  );
}
