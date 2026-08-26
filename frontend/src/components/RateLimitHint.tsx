import {useNavigate} from 'react-router-dom';
import {AlertTriangle, LogIn} from 'lucide-react';

// 判断错误信息是否为 GitHub API 次数用尽（rate limit）
export const isRateLimitMessage = (msg: string): boolean =>
    /rate limit|rate.?limit|403|exceeded|配额|已用尽/i.test(msg || '');

// 格式化时间（RFC3339 → 本地 YYYY-MM-DD HH:mm）
export const formatTime = (value?: string): string => {
    if (!value) return '';
    const d = new Date(value);
    if (isNaN(d.getTime())) return '';
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
};

// 额度用完时的引导提示：未授权则引导去设置页授权
export function RateLimitHint({message, authenticated}: {message: string; authenticated: boolean}) {
    const navigate = useNavigate();
    if (!isRateLimitMessage(message)) return null;
    return (
        <div className="mt-3 rounded-md bg-amber-50 p-3 text-sm text-amber-800">
            <div className="flex items-start gap-2">
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0"/>
                <div className="flex-1">
                    <p className="font-medium">GitHub 额度已用完</p>
                    {authenticated ? (
                        <p className="mt-1 text-xs">
                            本小时的 API 调用已达到上限，可稍后重试；若持续出现，请前往设置页检查授权信息。
                        </p>
                    ) : (
                        <p className="mt-1 text-xs">
                            当前未登录 GitHub，未授权时额度很低、很容易用完。建议前往设置页登录授权，获得更充足的专属额度。
                        </p>
                    )}
                </div>
                {!authenticated && (
                    <button
                        onClick={() => navigate('/settings')}
                        className="flex shrink-0 items-center gap-1.5 rounded-md bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700"
                    >
                        <LogIn className="h-3.5 w-3.5"/>
                        去授权
                    </button>
                )}
            </div>
        </div>
    );
}
