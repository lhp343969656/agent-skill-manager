import {create} from 'zustand';
import {GetAppInfo, ListSkills, ListAgents, DetectAgents, SetAgentEnabled, SetAgentSkillsPath, PickAgentSkillsDir, SetSharedDir, SetDownloadMirror, SetGitHubToken, StartGitHubOAuth, CheckGitHubOAuth, ClearGitHubAuth, GetGitHubRateLimit, UpdateSkillNote} from '../../wailsjs/go/main/App';
export interface OAuthInfo {
    verificationUri: string;
    userCode: string;
    deviceCode: string;
}

export interface SkillItem {
    id: string;
    displayName: string;
    sourceUrl: string;
    repositoryOwner: string;
    repositoryName: string;
    repositoryPath: string;
    currentVersionId: string;
    displayVersion: string;
    note: string;
    installedAt: string;
}

export interface AgentItem {
    id: string;
    adapterId: string;
    displayName: string;
    skillsPath: string;
    enabled: boolean;
    detected: boolean;
}

export interface RateLimitInfo {
    limit: number;
    remaining: number;
    resetUnix: number;
    resetTime: string;
    resetMinutes: number;
    authenticated: boolean;
}

interface AppState {
    appName: string;
    version: string;
    goos: string;
    configDir: string;
    sharedDir: string;
    downloadMirror: string;
    isConfigured: boolean;
    gitHubAuthMethod: string;
    gitHubTokenSet: boolean;
    gitHubUserLogin: string;
    gitHubUserName: string;
    githubRateLimit: RateLimitInfo | null;
    skills: SkillItem[];
    agents: AgentItem[];
    loading: boolean;
    error: string | null;

    init: () => Promise<void>;
    refreshSkills: () => Promise<void>;
    refreshAgents: () => Promise<void>;
    detectAgents: () => Promise<void>;
    updateAgentEnabled: (id: string, enabled: boolean) => Promise<void>;
    updateAgentSkillsPath: (id: string, path: string) => Promise<void>;
    pickAgentSkillsDir: () => Promise<string>;
    setSharedDir: (path: string) => Promise<void>;
    setDownloadMirror: (mirror: string) => Promise<void>;
    setGitHubToken: (token: string) => Promise<void>;
    refreshGitHubRateLimit: () => Promise<void>;
    oauthInfo: OAuthInfo | null;
    oauthLoading: boolean;
    startGitHubOAuth: () => Promise<OAuthInfo>;
    checkGitHubOAuth: (deviceCode: string) => Promise<void>;
    clearGitHubAuth: () => Promise<void>;
    updateSkillNote: (skillId: string, note: string) => Promise<void>;
}

export const useAppStore = create<AppState>((set) => ({
    appName: '',
    version: '',
    goos: '',
    configDir: '',
    sharedDir: '',
    downloadMirror: '',
    isConfigured: false,
    gitHubAuthMethod: '',
    gitHubTokenSet: false,
    gitHubUserLogin: '',
    gitHubUserName: '',
    githubRateLimit: null,
    skills: [],
    agents: [],
    loading: false,
    error: null,
    oauthInfo: null,
    oauthLoading: false,

    init: async () => {
        try {
            const info = await GetAppInfo();
            set({
                appName: info.name || '',
                version: info.version || '',
                goos: info.goos || '',
                configDir: info.configDir || '',
                sharedDir: info.sharedDir || '',
                downloadMirror: info.downloadMirror || '',
                isConfigured: info.isConfigured === 'true',
                gitHubAuthMethod: info.gitHubAuthMethod || '',
                gitHubTokenSet: info.gitHubTokenSet === 'true',
                gitHubUserLogin: info.gitHubUserLogin || '',
                gitHubUserName: info.gitHubUserName || '',
            });
            // 刷新剩余额度（不阻塞主流程；授权/解除授权/填 token 后 init 调用会同步刷新）
            useAppStore.getState().refreshGitHubRateLimit();
            if (info.isConfigured === 'true') {
                await Promise.all([useAppStore.getState().refreshSkills(), useAppStore.getState().refreshAgents()]);
            }
        } catch (e: any) {
            set({error: e?.message || String(e)});
        }
    },

    refreshSkills: async () => {
        try {
            set({loading: true});
            const skills = await ListSkills();
            set({skills, loading: false, error: null});
        } catch (e: any) {
            set({loading: false, error: e?.message || String(e)});
        }
    },

    refreshAgents: async () => {
        try {
            set({loading: true});
            const agents = await ListAgents();
            set({agents, loading: false, error: null});
        } catch (e: any) {
            set({loading: false, error: e?.message || String(e)});
        }
    },

    detectAgents: async () => {
        try {
            set({loading: true});
            const agents = await DetectAgents();
            set({agents, loading: false, error: null});
        } catch (e: any) {
            set({loading: false, error: e?.message || String(e)});
        }
    },

    updateAgentEnabled: async (id: string, enabled: boolean) => {
        try {
            set({loading: true});
            const agents = await SetAgentEnabled(id, enabled);
            set({agents, loading: false, error: null});
        } catch (e: any) {
            // 启用可能已生效（如部分技能同步失败时后端仍会启用），刷新列表以反映真实状态
            try {
                const agents = await ListAgents();
                set({agents, loading: false, error: e?.message || String(e)});
            } catch {
                set({loading: false, error: e?.message || String(e)});
            }
        }
    },

    updateAgentSkillsPath: async (id: string, path: string) => {
        try {
            set({loading: true});
            const agents = await SetAgentSkillsPath(id, path);
            set({agents, loading: false, error: null});
        } catch (e: any) {
            set({loading: false, error: e?.message || String(e)});
        }
    },

    pickAgentSkillsDir: async () => {
        try {
            return await PickAgentSkillsDir();
        } catch (e: any) {
            set({error: e?.message || String(e)});
            return '';
        }
    },

    setSharedDir: async (path: string) => {
        await SetSharedDir(path);
        await useAppStore.getState().init();
    },

    setDownloadMirror: async (mirror: string) => {
        await SetDownloadMirror(mirror.trim());
        await useAppStore.getState().init();
    },

    setGitHubToken: async (token: string) => {
        await SetGitHubToken(token.trim());
        await useAppStore.getState().init();
    },

    refreshGitHubRateLimit: async () => {
        try {
            const info = await GetGitHubRateLimit();
            set({githubRateLimit: info as RateLimitInfo});
        } catch (e: any) {
            set({githubRateLimit: null});
        }
    },

    startGitHubOAuth: async () => {
        set({oauthLoading: true, oauthInfo: null});
        try {
            const res = await StartGitHubOAuth();
            const info: OAuthInfo = {verificationUri: res.verificationUri, userCode: res.userCode, deviceCode: res.deviceCode};
            set({oauthInfo: info, oauthLoading: false});
            return info;
        } catch (e: any) {
            set({oauthLoading: false});
            throw e;
        }
    },

    checkGitHubOAuth: async (deviceCode: string) => {
        set({oauthLoading: true});
        try {
            await CheckGitHubOAuth(deviceCode);
            await useAppStore.getState().init();
            set({oauthInfo: null, oauthLoading: false});
        } catch (e: any) {
            set({oauthLoading: false});
            throw e;
        }
    },

    clearGitHubAuth: async () => {
        await ClearGitHubAuth();
        await useAppStore.getState().init();
    },

    updateSkillNote: async (skillId: string, note: string) => {
        await UpdateSkillNote(skillId, note.trim());
        await useAppStore.getState().refreshSkills();
    },
}));
