import React from 'react';

interface Props {
    children: React.ReactNode;
}

interface State {
    hasError: boolean;
    error: Error | null;
}

// ErrorBoundary 捕获子组件渲染异常，显示错误信息而非白屏
export class ErrorBoundary extends React.Component<Props, State> {
    constructor(props: Props) {
        super(props);
        this.state = {hasError: false, error: null};
    }

    static getDerivedStateFromError(error: Error): State {
        return {hasError: true, error};
    }

    componentDidCatch(error: Error, info: React.ErrorInfo) {
        console.error('渲染错误:', error, info);
    }

    render() {
        if (this.state.hasError) {
            return (
                <div className="flex h-full items-center justify-center p-8">
                    <div className="max-w-md rounded-lg border border-red-200 bg-red-50 p-6">
                        <h2 className="text-lg font-bold text-red-800">页面出现了问题</h2>
                        <p className="mt-2 break-all text-sm text-red-700">{this.state.error?.message || '未知错误'}</p>
                        <button
                            onClick={() => window.location.reload()}
                            className="mt-4 rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
                        >
                            重新加载
                        </button>
                    </div>
                </div>
            );
        }
        return this.props.children;
    }
}
