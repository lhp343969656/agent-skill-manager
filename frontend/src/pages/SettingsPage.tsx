import {useEffect, useState} from 'react';
import {FolderOpen, Save, RefreshCw, Gauge, ShieldCheck, ShieldAlert, LogOut, GitBranch, KeyRound, ExternalLink, AlertTriangle} from 'lucide-react';
import {useAppStore} from '../stores/appStore';

export function SettingsPage() {
    const {
        sharedDir, configDir, downloadMirror, isConfigured,
        gitHubAuthMethod, gitHubTokenSet, gitHubUserLogin, gitHubUserName,
        githubRateLimit, refreshGitHubRateLimit,
        setSharedDir, setDownloadMirror,
        setGitHubToken, startGitHubOAuth, checkGitHubOAuth, clearGitHubAuth,
        oauthInfo, oauthLoading,
    } = useAppStore();
    const [dirInput, setDirInput] = useState(sharedDir);
    const [saving, setSaving] = useState(false);
    const [message, setMessage] = useState<string | null>(null);
    const [error, setError] = useState<string | null>(null);

    // 镜像配置
    const [mirrorInput, setMirrorInput] = useState(downloadMirror);
    const [savingMirror, setSavingMirror] = useState(false);
    const [mirrorMsg, setMirrorMsg] = useState<string | null>(null);
    const [mirrorErr, setMirrorErr] = useState<string | null>(null);

    // GitHub 授权
    const [tokenInput, setTokenInput] = useState('');
    const [savingToken, setSavingToken] = useState(false);
    const [authMsg, setAuthMsg] = useState<string | null>(null);
    const [authErr, setAuthErr] = useState<string | null>(null);
    // 授权按钮区域的醒目错误（用于网络不通等需要明显提示的场景）
    const [oauthErr, setOauthErr] = useState<string | null>(null);

    const isAuthorized = gitHubTokenSet;

    const handleSave = async () => {
        setSaving(true);
        setError(null);
        setMessage(null);
        try {
            await setSharedDir(dirInput.trim());
            setMessage('共享目录已保存并初始化数据库');
        } catch (e: any) {
            setError(e?.message || String(e));
        } finally {
            setSaving(false);
        }
    };

    const handleSaveMirror = async () => {
        setSavingMirror(true);
        setMirrorErr(null);
        setMirrorMsg(null);
        try {
            await setDownloadMirror(mirrorInput.trim());
            setMirrorMsg(mirrorInput.trim() ? '下载加速已启用' : '已关闭下载加速');
        } catch (e: any) {
            setMirrorErr(e?.message || String(e));
        } finally {
            setSavingMirror(false);
        }
    };

    // 手动填写 token
    const handleSaveToken = async () => {
        if (tokenInput.trim() === '') return;
        setSavingToken(true);
        setAuthErr(null);
        setAuthMsg(null);
        try {
            await setGitHubToken(tokenInput.trim());
            setTokenInput('');
            setAuthMsg('已保存 token，检查更新/更新时将使用你的专属额度');
        } catch (e: any) {
            setAuthErr(e?.message || String(e));
        } finally {
            setSavingToken(false);
        }
    };

    // 登录授权（设备流）
    const handleStartOAuth = async () => {
        setAuthErr(null);
        setAuthMsg(null);
        setOauthErr(null);
        try {
            const info = await startGitHubOAuth();
            // 打开浏览器后，自动轮询等待授权结果，无需用户回应用再点「我已完成授权」
            await pollOAuth(info.deviceCode);
        } catch (e: any) {
            const msg = e?.message || String(e);
            // 在授权按钮区域就地显示醒目错误，避免被忽略
            setOauthErr(msg);
            setAuthErr(msg);
        }
    };

    const pollOAuth = async (deviceCode: string) => {
        try {
            await checkGitHubOAuth(deviceCode);
            setAuthMsg('授权成功，已开启专属额度');
        } catch (e: any) {
            setAuthErr(e?.message || String(e));
        }
    };

    // 手动刷新授权结果（兜底：自动轮询异常或超时后可用）
    const handlePollOAuth = async () => {
        if (!oauthInfo) return;
        setAuthErr(null);
        await pollOAuth(oauthInfo.deviceCode);
    };

    // 清除授权
    const handleClearAuth = async () => {
        setAuthErr(null);
        setAuthMsg(null);
        try {
            await clearGitHubAuth();
            setAuthMsg('已清除授权，恢复为受限模式');
        } catch (e: any) {
            setAuthErr(e?.message || String(e));
        }
    };

    // 复制验证码
    const handleCopyCode = async () => {
        if (!oauthInfo) return;
        try {
            await navigator.clipboard.writeText(oauthInfo.userCode);
            setAuthErr(null);
            setAuthMsg('验证码已复制，请到浏览器页面粘贴输入');
        } catch (e: any) {
            setAuthErr('复制失败，请手动输入验证码');
        }
    };

    // 剩余额度：进入页面自动拉取一次，可手动刷新
    const [rateLimitLoading, setRateLimitLoading] = useState(false);
    const handleRefreshRateLimit = async () => {
        setRateLimitLoading(true);
        try {
            await refreshGitHubRateLimit();
        } finally {
            setRateLimitLoading(false);
        }
    };
    useEffect(() => {
        refreshGitHubRateLimit();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const formatReset = (minutes: number) => {
        if (minutes <= 0) return '即将';
        if (minutes < 60) return `约 ${minutes} 分钟`;
        return `约 ${Math.floor(minutes / 60)} 小时`;
    };

    // 进度条：剩余占比与颜色（绿色充足、黄色中等、红色快用完）
    const ratePct = githubRateLimit && githubRateLimit.limit > 0
        ? Math.min(100, Math.round((githubRateLimit.remaining / githubRateLimit.limit) * 100))
        : 0;
    const rateBarColor = ratePct >= 50 ? 'bg-green-500' : ratePct >= 20 ? 'bg-yellow-500' : 'bg-red-500';
    const rateTextColor = ratePct >= 50 ? 'text-green-600' : ratePct >= 20 ? 'text-yellow-600' : 'text-red-600';

    return (
        <div className="p-6 max-w-2xl">
            <h1 className="text-2xl font-bold text-gray-900">设置</h1>
            <p className="mt-1 text-sm text-gray-500">管理共享 Skill 目录与应用配置</p>

            {/* 共享目录设置 */}
            <section className="mt-8">
                <h2 className="mb-2 text-sm font-semibold text-gray-900">共享 Skill 目录</h2>
                <p className="mb-3 text-xs text-gray-500">
                    所有共享 Skill 的存放位置。修改后会在此目录初始化数据库。
                </p>
                <div className="flex gap-2">
                    <div className="relative flex-1">
                        <FolderOpen className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-gray-400"/>
                        <input
                            value={dirInput}
                            onChange={(e) => setDirInput(e.target.value)}
                            placeholder="例如：D:\AgentSkills"
                            className="w-full rounded-md border border-gray-300 bg-white py-2 pl-9 pr-3 text-sm text-gray-900 placeholder-gray-400 focus:border-indigo-500 focus:outline-none"
                        />
                    </div>
                    <button
                        onClick={handleSave}
                        disabled={saving || !dirInput.trim()}
                        className="flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:bg-gray-300"
                    >
                        <Save className="h-4 w-4"/>
                        {saving ? '保存中...' : '保存'}
                    </button>
                </div>
                {message && <p className="mt-2 text-sm text-green-600">{message}</p>}
                {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
            </section>

            {/* GitHub 下载加速 */}
            <section className="mt-8">
                <h2 className="mb-2 text-sm font-semibold text-gray-900">GitHub 下载加速</h2>
                <p className="mb-3 text-xs text-gray-500">
                    访问 GitHub 下载较慢时，可配置一个加速镜像前缀（如{' '}
                    <code className="rounded bg-gray-100 px-1">https://ghproxy.com/</code>
                    ）。下载 Skill 时会自动通过镜像加速。留空则直接连接 GitHub。
                </p>
                <div className="flex gap-2">
                    <div className="relative flex-1">
                        <Gauge className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-gray-400"/>
                        <input
                            value={mirrorInput}
                            onChange={(e) => setMirrorInput(e.target.value)}
                            placeholder="例如：https://ghproxy.com/"
                            className="w-full rounded-md border border-gray-300 bg-white py-2 pl-9 pr-3 text-sm text-gray-900 placeholder-gray-400 focus:border-indigo-500 focus:outline-none"
                        />
                    </div>
                    <button
                        onClick={handleSaveMirror}
                        disabled={savingMirror}
                        className="flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:bg-gray-300"
                    >
                        <Save className="h-4 w-4"/>
                        {savingMirror ? '保存中...' : '保存'}
                    </button>
                </div>
                {mirrorMsg && <p className="mt-2 text-sm text-green-600">{mirrorMsg}</p>}
                {mirrorErr && <p className="mt-2 text-sm text-red-600">{mirrorErr}</p>}
            </section>

            {/* GitHub 授权 */}
            <section className="mt-8 rounded-lg border border-gray-200 bg-white p-5">
                <div className="flex items-center gap-2">
                    <GitBranch className="h-5 w-5 text-gray-700"/>
                    <h2 className="text-sm font-semibold text-gray-900">GitHub 授权（提升访问额度）</h2>
                </div>
                <p className="mt-2 text-xs text-gray-500">
                    GitHub 公开接口每小时有 60 次限制（按共享 IP 计算，容易撞上限）。授权后额度提升到
                    5000 次/小时，检查更新/更新更稳定。授权方式二选一，可随时修改。
                </p>

                {/* 状态提示 */}
                <div className={`mt-4 flex items-start gap-2 rounded-md p-3 text-sm ${
                    isAuthorized ? 'bg-green-50 text-green-700' : 'bg-yellow-50 text-yellow-700'
                }`}>
                    {isAuthorized ? <ShieldCheck className="mt-0.5 h-4 w-4 flex-shrink-0"/> : <ShieldAlert className="mt-0.5 h-4 w-4 flex-shrink-0"/>}
                    <span>
                        {isAuthorized
                            ? '当前已授权，使用专属额度（5000 次/小时）。'
                            : '当前为受限模式（60 次/小时），频繁操作可能遇到访问限制。可授权解除。'}
                    </span>
                </div>

                {/* 剩余额度 */}
                <div className="mt-3 rounded-md border border-gray-200 bg-gray-50 p-3">
                    <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                            <Gauge className="h-4 w-4 text-gray-500"/>
                            <span className="text-sm text-gray-700">本小时剩余额度</span>
                            <span className={`text-xs font-medium ${rateTextColor}`}>
                                {githubRateLimit ? (githubRateLimit.authenticated ? '已授权' : '未授权') : ''}
                            </span>
                        </div>
                        <button
                            onClick={handleRefreshRateLimit}
                            disabled={rateLimitLoading}
                            className="inline-flex items-center gap-1.5 rounded-md border border-gray-300 bg-white px-2.5 py-1.5 text-xs text-gray-600 hover:bg-gray-50 disabled:opacity-50"
                        >
                            <RefreshCw className={`h-3.5 w-3.5 ${rateLimitLoading ? 'animate-spin' : ''}`}/>
                            刷新
                        </button>
                    </div>
                    {githubRateLimit ? (
                        <>
                            <div className="mt-2.5 flex items-center gap-3">
                                <div className="h-2 flex-1 overflow-hidden rounded-full bg-gray-200">
                                    <div
                                        className={`h-full rounded-full transition-all ${rateBarColor}`}
                                        style={{width: `${ratePct}%`}}
                                    />
                                </div>
                                <span className="text-sm font-semibold text-gray-900">{githubRateLimit.remaining}</span>
                                <span className="text-xs text-gray-400">/ {githubRateLimit.limit} 次</span>
                            </div>
                            <p className="mt-1.5 text-xs text-gray-400">
                                已用 {githubRateLimit.limit - githubRateLimit.remaining} 次 ·{' '}
                                {formatReset(githubRateLimit.resetMinutes)} 后重置
                            </p>
                        </>
                    ) : (
                        <p className="mt-2 text-sm text-gray-500">暂无剩余额度数据，点右侧「刷新」查看</p>
                    )}
                </div>

                {/* 已授权：显示账号信息 + 解除授权 */}
                {isAuthorized && (
                    <div className="mt-4 flex items-center justify-between rounded-md border border-green-200 bg-green-50 p-3">
                        <div className="flex items-center gap-3">
                            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-green-600 text-sm font-bold text-white">
                                {(gitHubUserLogin || '?').charAt(0).toUpperCase()}
                            </div>
                            <div>
                                <p className="text-sm font-medium text-gray-900">
                                    {gitHubUserLogin || 'GitHub 账号'}
                                </p>
                                <p className="text-xs text-gray-500">
                                    授权方式：{gitHubAuthMethod === 'token' ? '手动 token' : '登录授权'}
                                    {gitHubUserName ? ` · ${gitHubUserName}` : ''}
                                </p>
                            </div>
                        </div>
                        <button
                            onClick={handleClearAuth}
                            className="inline-flex items-center gap-1.5 rounded-md border border-red-300 bg-white px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50"
                        >
                            <LogOut className="h-3.5 w-3.5"/>
                            解除授权
                        </button>
                    </div>
                )}

                {!isAuthorized && (<>
                {/* 方式一：手动填写 token */}
                <div className="mt-4">
                    <div className="flex items-center gap-2 text-sm font-medium text-gray-700">
                        <KeyRound className="h-4 w-4 text-gray-500"/>
                        方式一：手动填写 token
                    </div>
                    <p className="mt-1 text-xs text-gray-500">
                        在 GitHub → Settings → Developer settings → Personal access tokens 生成，勾选{' '}
                        <code className="rounded bg-gray-100 px-1">repo</code> 权限，粘贴到下方。
                    </p>
                    <div className="mt-2 flex gap-2">
                        <input
                            type="password"
                            value={tokenInput}
                            onChange={(e) => setTokenInput(e.target.value)}
                            placeholder="粘贴 ghp_ 开头的 token"
                            className="flex-1 rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-indigo-500 focus:outline-none"
                        />
                        <button
                            onClick={handleSaveToken}
                            disabled={savingToken || !tokenInput.trim()}
                            className="inline-flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:bg-gray-300"
                        >
                            <Save className="h-4 w-4"/>
                            {savingToken ? '保存中...' : '保存'}
                        </button>
                    </div>
                    <p className="mt-1 text-xs text-gray-400">token 仅存在你本机，不会上传。</p>
                </div>

                {/* 方式二：登录授权 */}
                <div className="mt-5">
                    <div className="flex items-center gap-2 text-sm font-medium text-gray-700">
                        <GitBranch className="h-4 w-4 text-gray-500"/>
                        方式二：用 GitHub 账号登录授权
                    </div>
                    <p className="mt-1 text-xs text-gray-500">
                        点「开始授权」后会打开浏览器，你确认后自动完成，无需复制粘贴。适合不想手动操作的用户。
                    </p>
                    {!oauthInfo ? (
                        <>
                        <button
                            onClick={handleStartOAuth}
                            disabled={oauthLoading}
                            className="mt-2 inline-flex items-center gap-2 rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
                        >
                            <GitBranch className="h-4 w-4"/>
                            {oauthLoading ? '授权中...' : '开始授权'}
                        </button>
                        {oauthErr && (
                            <div className="mt-3 flex items-start gap-2 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
                                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0"/>
                                <div>
                                    <p className="font-semibold">GitHub 授权失败</p>
                                    <p className="mt-1 text-xs leading-relaxed text-red-600">{oauthErr}</p>
                                    <p className="mt-1 text-xs text-red-500">
                                        提示：若提示无法连接 GitHub，请检查网络或代理后重试。
                                    </p>
                                </div>
                            </div>
                        )}
                        </>
                    ) : (
                        <div className="mt-3 rounded-lg border-2 border-indigo-300 bg-indigo-50 p-4">
                            <p className="text-sm font-medium text-indigo-900">已为你打开 GitHub 授权页，请完成下面两步</p>
                            <div className="mt-3 rounded-lg border border-indigo-200 bg-white p-4 text-center">
                                <p className="text-xs text-gray-500">第一步：在新打开的页面里，输入下面这个验证码</p>
                                <div className="mt-2 inline-flex items-center gap-3">
                                    <span className="font-mono text-3xl font-bold tracking-wider text-indigo-700">{oauthInfo.userCode}</span>
                                    <button
                                        onClick={handleCopyCode}
                                        title="复制验证码"
                                        className="rounded-md border border-gray-300 bg-white px-2 py-1 text-xs text-gray-600 hover:bg-gray-50"
                                    >
                                        复制
                                    </button>
                                </div>
                            </div>
                            <p className="mt-3 text-xs text-gray-500">
                                第二步：在浏览器完成授权后，应用会自动检测到并更新，无需手动操作
                                {oauthInfo.verificationUri && (<> · 若浏览器没自动弹出，可手动访问{' '}
                                    <a
                                        href={oauthInfo.verificationUri}
                                        target="_blank"
                                        rel="noreferrer"
                                        className="inline-flex items-center gap-1 text-indigo-600 hover:underline"
                                    >
                                        GitHub 授权页 <ExternalLink className="h-3 w-3"/>
                                    </a>
                                </>)}
                            </p>
                            <div className="mt-3 inline-flex w-full items-center justify-center gap-2 rounded-md bg-indigo-50 px-4 py-2 text-sm font-medium text-indigo-700">
                                <RefreshCw className={`h-4 w-4 ${oauthLoading ? 'animate-spin' : ''}`}/>
                                {oauthLoading ? '正在等待授权完成...' : '等待你在浏览器确认授权'}
                            </div>
                            {!oauthLoading && (
                                <button
                                    onClick={handlePollOAuth}
                                    className="mt-2 inline-flex w-full items-center justify-center gap-2 rounded-md border border-indigo-300 px-4 py-2 text-sm font-medium text-indigo-700 hover:bg-indigo-100"
                                >
                                    我已确认，手动检查一次
                                </button>
                            )}
                        </div>
                    )}
                </div>
                </>)}

                {authMsg && <p className="mt-3 text-sm text-green-600">{authMsg}</p>}
                {authErr && <p className="mt-3 text-sm text-red-600">{authErr}</p>}
            </section>

            {/* 当前配置信息 */}
            <section className="mt-8">
                <h2 className="mb-2 text-sm font-semibold text-gray-900">当前配置</h2>
                <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
                    <dl className="divide-y divide-gray-100">
                        <div className="flex px-4 py-3">
                            <dt className="w-32 text-sm text-gray-500">共享目录</dt>
                            <dd className="text-sm text-gray-900">{sharedDir || '（未配置）'}</dd>
                        </div>
                        <div className="flex px-4 py-3">
                            <dt className="w-32 text-sm text-gray-500">下载加速</dt>
                            <dd className="text-sm text-gray-900">{downloadMirror || '（未启用）'}</dd>
                        </div>
                        <div className="flex px-4 py-3">
                            <dt className="w-32 text-sm text-gray-500">应用配置目录</dt>
                            <dd className="text-sm text-gray-900">{configDir}</dd>
                        </div>
                        <div className="flex px-4 py-3">
                            <dt className="w-32 text-sm text-gray-500">状态</dt>
                            <dd className="text-sm text-gray-900">
                                <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                                    isConfigured ? 'bg-green-50 text-green-700' : 'bg-yellow-50 text-yellow-700'
                                }`}>
                                    {isConfigured ? '已配置' : '未配置'}
                                </span>
                            </dd>
                        </div>
                    </dl>
                </div>
            </section>

            {/* 占位功能 */}
            <section className="mt-8 rounded-lg border border-gray-200 bg-white p-6">
                <h2 className="mb-2 text-sm font-semibold text-gray-900">即将提供</h2>
                <ul className="space-y-2 text-sm text-gray-500">
                    <li className="flex items-center gap-2">
                        <RefreshCw className="h-4 w-4 text-gray-400"/>
                        迁移共享目录（阶段三实现）
                    </li>
                    <li className="flex items-center gap-2">
                        <FolderOpen className="h-4 w-4 text-gray-400"/>
                        缓存与日志管理（阶段三实现）
                    </li>
                </ul>
            </section>
        </div>
    );
}
