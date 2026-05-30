import { useEffect, useMemo, useState } from 'react'
import clsx from 'clsx'

import { useI18n } from '../../i18n'
import { useApiTesterState } from './useApiTesterState'
import { useChatStreamClient } from './useChatStreamClient'
import ConfigPanel from './ConfigPanel'
import ChatPanel from './ChatPanel'

function describeModel(t, modelID) {
    switch (modelID) {
    case 'ernie-4.5-turbo':
        return t('apiTester.models.ernie')
    case 'deepseek-v4-pro':
        return t('apiTester.models.dsv4pro')
    case 'deepseek-r1':
        return t('apiTester.models.dsr1')
    case 'smartmode':
        return t('apiTester.models.smart')
    case 'smartmode-thinking':
        return t('apiTester.models.smartThinking')
    default:
        return t('apiTester.models.generic')
    }
}

function decorateModel(t, modelID) {
    switch (modelID) {
    case 'ernie-4.5-turbo':
        return { id: modelID, name: 'ERNIE 4.5 Turbo', icon: 'Zap', desc: describeModel(t, modelID), color: 'text-blue-600' }
    case 'deepseek-v4-pro':
        return { id: modelID, name: 'DeepSeek V4 Pro', icon: 'Cpu', desc: describeModel(t, modelID), color: 'text-amber-600' }
    case 'deepseek-r1':
        return { id: modelID, name: 'DeepSeek R1', icon: 'Terminal', desc: describeModel(t, modelID), color: 'text-violet-600' }
    case 'smartmode':
        return { id: modelID, name: 'SmartMode', icon: 'MessageSquare', desc: describeModel(t, modelID), color: 'text-emerald-600' }
    case 'smartmode-thinking':
        return { id: modelID, name: 'SmartMode Thinking', icon: 'SearchIcon', desc: describeModel(t, modelID), color: 'text-cyan-600' }
    default:
        return { id: modelID, name: modelID, icon: 'MessageSquare', desc: describeModel(t, modelID), color: 'text-amber-500' }
    }
}

export default function ApiTesterContainer({ config, onMessage, authFetch }) {
    const { t } = useI18n()
    const [availableModelIDs, setAvailableModelIDs] = useState([])
    const [modelsLoaded, setModelsLoaded] = useState(false)

    const {
        model,
        setModel,
        message,
        setMessage,
        attachedFiles,
        setAttachedFiles,
        apiKey,
        setApiKey,
        selectedAccount,
        setSelectedAccount,
        response,
        setResponse,
        loading,
        setLoading,
        streamingContent,
        setStreamingContent,
        streamingThinking,
        setStreamingThinking,
        isStreaming,
        setIsStreaming,
        streamingMode,
        setStreamingMode,
        configExpanded,
        setConfigExpanded,
        abortControllerRef,
    } = useApiTesterState({ t })

    const accounts = config.accounts || []
    const resolveAccountIdentifier = (acc) => {
        if (!acc || typeof acc !== 'object') return ''
        return String(acc.identifier || acc.email || acc.mobile || '').trim()
    }
    const configuredKeys = config.keys || []
    const trimmedApiKey = apiKey.trim()
    const defaultKey = configuredKeys[0] || ''
    const effectiveKey = trimmedApiKey || defaultKey
    const customKeyActive = trimmedApiKey !== ''
    const customKeyManaged = customKeyActive && configuredKeys.includes(trimmedApiKey)

    useEffect(() => {
        let disposed = false

        async function loadModels() {
            try {
                const res = await authFetch('/v1/models')
                if (!res.ok) {
                    throw new Error(`failed to fetch models: ${res.status}`)
                }
                const data = await res.json()
                const modelIDs = Array.isArray(data?.data)
                    ? data.data
                        .map((item) => String(item?.id || '').trim())
                        .filter(Boolean)
                    : []
                if (!disposed) {
                    setAvailableModelIDs(modelIDs)
                }
            } catch (_err) {
                if (!disposed) {
                    setAvailableModelIDs([])
                }
            } finally {
                if (!disposed) {
                    setModelsLoaded(true)
                }
            }
        }

        setModelsLoaded(false)
        loadModels()
        return () => {
            disposed = true
        }
    }, [authFetch])

    const models = useMemo(
        () => availableModelIDs.map((modelID) => decorateModel(t, modelID)),
        [availableModelIDs, t]
    )

    useEffect(() => {
        if (!models.length) {
            if (model) {
                setModel('')
            }
            return
        }
        if (!model || !models.some((item) => item.id === model)) {
            setModel(models[0].id)
        }
    }, [model, models, setModel])

    const { runTest, stopGeneration } = useChatStreamClient({
        t,
        onMessage,
        model,
        message,
        effectiveKey,
        selectedAccount,
        streamingMode,
        attachedFiles,
        abortControllerRef,
        setLoading,
        setIsStreaming,
        setResponse,
        setStreamingContent,
        setStreamingThinking,
    })

    return (
        <div className={clsx('flex flex-col lg:grid lg:grid-cols-12 gap-6 h-[calc(100vh-140px)] min-h-0')}>
            <ConfigPanel
                t={t}
                configExpanded={configExpanded}
                setConfigExpanded={setConfigExpanded}
                models={models}
                model={model}
                setModel={setModel}
                modelsLoaded={modelsLoaded}
                streamingMode={streamingMode}
                setStreamingMode={setStreamingMode}
                selectedAccount={selectedAccount}
                setSelectedAccount={setSelectedAccount}
                accounts={accounts}
                resolveAccountIdentifier={resolveAccountIdentifier}
                apiKey={apiKey}
                setApiKey={setApiKey}
                config={config}
                customKeyActive={customKeyActive}
                customKeyManaged={customKeyManaged}
            />

            <ChatPanel
                t={t}
                message={message}
                setMessage={setMessage}
                attachedFiles={attachedFiles}
                setAttachedFiles={setAttachedFiles}
                setSelectedAccount={setSelectedAccount}
                effectiveKey={effectiveKey}
                selectedAccount={selectedAccount}
                model={model}
                onMessage={onMessage}
                response={response}
                isStreaming={isStreaming}
                loading={loading}
                streamingThinking={streamingThinking}
                streamingContent={streamingContent}
                onRunTest={runTest}
                onStopGeneration={stopGeneration}
                hasAvailableModel={models.length > 0}
            />
        </div>
    )
}
