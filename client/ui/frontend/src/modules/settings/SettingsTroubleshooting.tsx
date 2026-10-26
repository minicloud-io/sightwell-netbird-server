import { useId, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import {
    BookText,
    CircleCheckBig,
    FolderOpen,
    Info,
    Loader2,
    Shield,
    ShieldCheck,
    ShieldOff,
    type LucideIcon,
} from "lucide-react";
import { Browser } from "@wailsio/runtime";
import { Debug as DebugSvc } from "@bindings/services";
import type { DebugBundleResult } from "@bindings/services/models.js";
import { Button } from "@/components/buttons/Button";
import { DialogActions } from "@/components/dialog/DialogActions";
import { DialogDescription } from "@/components/dialog/DialogDescription";
import { DialogHeading } from "@/components/dialog/DialogHeading";
import FancyToggleSwitch from "@/components/switches/FancyToggleSwitch";
import HelpText from "@/components/typography/HelpText.tsx";
import { Input } from "@/components/inputs/Input";
import { Label } from "@/components/typography/Label";
import { Select } from "@/components/inputs/Select";
import { SquareIcon } from "@/components/SquareIcon";
import { Tooltip } from "@/components/Tooltip";
import { formatRemaining } from "@/lib/formatters";
import type { AnonymizeLevel, DebugStage } from "@/contexts/DebugBundleContext";
import { useDebugBundleContext } from "@/contexts/DebugBundleContext";
import { SectionGroup, SettingsBottomBar } from "@/modules/settings/SettingsSection.tsx";
// Sightwell fork: bundles are saved locally only (no upload to NetBird's
// service); the done screen links our troubleshooting docs (i18n/brand.json).
import brand from "../../../../i18n/brand.json";

const TROUBLESHOOTING_DOCS_URL = brand.links.troubleshooting;

const ANONYMIZE_LEVELS: { value: AnonymizeLevel; icon: LucideIcon }[] = [
    { value: "none", icon: ShieldOff },
    { value: "default", icon: Shield },
    { value: "strict", icon: ShieldCheck },
];

export function SettingsTroubleshooting() {
    const { t } = useTranslation();
    const durationId = useId();
    const {
        anonymizeLevel,
        setAnonymizeLevel,
        systemInfo,
        setSystemInfo,
        trace,
        setTrace,
        capture,
        setCapture,
        traceMinutes,
        setTraceMinutes,
        capturePackets,
        setCapturePackets,
        run,
        stage,
        cancel,
        reset,
    } = useDebugBundleContext();

    if (stage.kind === "done") {
        return <DoneResult result={stage.result} onClose={reset} />;
    }
    if (stage.kind !== "idle") {
        return <ProgressSection stage={stage} onCancel={cancel} />;
    }

    return (
        <SectionGroup title={t("settings.troubleshooting.section.title")}>
            <div className={"flex items-center justify-between gap-6"}>
                <div className={"max-w-md flex-1"}>
                    <Label as={"div"}>
                        <span className={"inline-flex items-center gap-1.5"}>
                            {t("settings.troubleshooting.anonymize.label")}
                            <Tooltip
                                content={
                                    <div className={"max-w-xs whitespace-normal leading-relaxed"}>
                                        {t("settings.troubleshooting.anonymize.info")}
                                    </div>
                                }
                            >
                                <Info
                                    size={14}
                                    aria-label={t("settings.troubleshooting.anonymize.label")}
                                    className={"shrink-0 cursor-default text-nb-gray-400"}
                                />
                            </Tooltip>
                        </span>
                    </Label>
                    <HelpText margin={false}>
                        {t("settings.troubleshooting.anonymize.help")}
                    </HelpText>
                </div>
                <div className={"shrink-0"}>
                    <Select
                        value={anonymizeLevel}
                        options={ANONYMIZE_LEVELS.map(({ value, icon }) => ({
                            value,
                            icon,
                            label: t(`settings.troubleshooting.anonymize.${value}`),
                        }))}
                        onChange={setAnonymizeLevel}
                        ariaLabel={t("settings.troubleshooting.anonymize.label")}
                    />
                </div>
            </div>
            <FancyToggleSwitch
                value={systemInfo}
                onChange={setSystemInfo}
                label={t("settings.troubleshooting.systemInfo.label")}
                helpText={t("settings.troubleshooting.systemInfo.help")}
            />
            <FancyToggleSwitch
                value={trace}
                onChange={setTrace}
                label={t("settings.troubleshooting.trace.label")}
                helpText={t("settings.troubleshooting.trace.help")}
            />
            <FancyToggleSwitch
                value={capture}
                onChange={setCapture}
                label={t("settings.troubleshooting.capture.label")}
                helpText={t("settings.troubleshooting.capture.help")}
            />
            <div className={"flex flex-col gap-4"}>
                <FancyToggleSwitch
                    value={capturePackets}
                    onChange={setCapturePackets}
                    label={t("settings.troubleshooting.packets.label")}
                    helpText={t("settings.troubleshooting.packets.help")}
                    disabled={!capture}
                />
                <div
                    className={"flex items-center justify-between gap-6"}
                    {...(capture ? {} : { inert: "" })}
                >
                    <div className={"max-w-md flex-1"}>
                        <Label htmlFor={durationId} disabled={!capture}>
                            {t("settings.troubleshooting.duration.label")}
                        </Label>
                        <HelpText margin={false} disabled={!capture}>
                            {t("settings.troubleshooting.duration.help")}
                        </HelpText>
                    </div>
                    <div className={"w-40 shrink-0"}>
                        <Input
                            id={durationId}
                            type={"number"}
                            min={1}
                            max={30}
                            value={traceMinutes}
                            onChange={(e) =>
                                setTraceMinutes(
                                    Math.max(1, Math.min(30, Number(e.target.value) || 1)),
                                )
                            }
                            customSuffix={t("settings.troubleshooting.duration.suffix")}
                            disabled={!capture}
                        />
                    </div>
                </div>
            </div>

            <SettingsBottomBar>
                <Button variant={"primary"} size={"md"} onClick={run}>
                    {t("settings.troubleshooting.create")}
                </Button>
            </SettingsBottomBar>
        </SectionGroup>
    );
}

function CenteredPanel({ children }: Readonly<{ children: ReactNode }>) {
    return (
        <div
            className={
                "absolute inset-0 flex flex-col items-center justify-center gap-5 p-8 text-center"
            }
        >
            {children}
        </div>
    );
}

function ProgressSection({
    stage,
    onCancel,
}: Readonly<{ stage: DebugStage; onCancel: () => void }>) {
    const { t } = useTranslation();
    const cancelling = stage.kind === "cancelling";
    return (
        <CenteredPanel>
            <SquareIcon icon={Loader2} className={"[&_svg]:animate-spin"} />

            <div className={"flex max-w-sm flex-col items-center gap-2"}>
                <DialogHeading className={"text-balance"}>{stageLabel(stage, t)}</DialogHeading>
                <DialogDescription>
                    {t("settings.troubleshooting.progress.description")}
                </DialogDescription>
            </div>

            {stage.kind === "capturing" && (
                <div
                    className={
                        "font-mono text-2xl font-semibold tabular-nums tracking-wider text-nb-gray-50"
                    }
                    aria-live={"polite"}
                >
                    {formatRemaining(stage.remainingSec)}
                </div>
            )}

            <DialogActions className={"max-w-[220px]"}>
                <Button
                    autoFocus
                    variant={"secondary"}
                    size={"md"}
                    className={"w-full"}
                    onClick={onCancel}
                    disabled={cancelling}
                >
                    {t("common.cancel")}
                </Button>
            </DialogActions>
        </CenteredPanel>
    );
}

function DoneResult({
    result,
    onClose,
}: Readonly<{
    result: DebugBundleResult;
    onClose: () => void;
}>) {
    const { t } = useTranslation();
    const onRevealPath = () => {
        if (!result.path) return;
        DebugSvc.RevealFile(result.path).catch((err: unknown) =>
            console.error("reveal debug bundle file", err),
        );
    };
    const onOpenDocs = () => {
        Browser.OpenURL(TROUBLESHOOTING_DOCS_URL).catch(() =>
            globalThis.open(TROUBLESHOOTING_DOCS_URL, "_blank"),
        );
    };
    return (
        <CenteredPanel>
            <SquareIcon
                icon={CircleCheckBig}
                className={"bg-white dark:bg-nb-gray-920 [&_svg]:text-green-500"}
            />

            <div className={"flex max-w-sm flex-col items-center gap-2"}>
                <DialogHeading className={"text-balance"}>
                    {t("settings.troubleshooting.done.savedTitle")}
                </DialogHeading>
                <DialogDescription>
                    {t("settings.troubleshooting.done.savedDescription")}
                </DialogDescription>
            </div>

            {result.path && (
                <div className={"flex w-full max-w-sm flex-col gap-3"}>
                    <Input
                        dir={"ltr"}
                        value={result.path}
                        readOnly
                        aria-label={t("settings.troubleshooting.done.savedTitle")}
                        customSuffix={
                            <button
                                type={"button"}
                                onClick={onRevealPath}
                                className={
                                    "pointer-events-auto transition-all hover:text-nb-gray-50"
                                }
                                aria-label={t("settings.troubleshooting.done.openFileLocation")}
                            >
                                <FolderOpen size={16} aria-hidden={"true"} />
                            </button>
                        }
                    />
                </div>
            )}

            <DialogActions className={"max-w-[220px]"}>
                {result.path && (
                    <Button
                        autoFocus
                        variant={"primary"}
                        size={"md"}
                        className={"w-full"}
                        onClick={onRevealPath}
                    >
                        <FolderOpen size={14} aria-hidden={"true"} />
                        {t("settings.troubleshooting.done.openFolder")}
                    </Button>
                )}
                <Button variant={"secondary"} size={"md"} className={"w-full"} onClick={onOpenDocs}>
                    <BookText size={14} aria-hidden={"true"} />
                    {t("settings.about.community.documentation")}
                </Button>
                <Button variant={"secondary"} size={"md"} className={"w-full"} onClick={onClose}>
                    {t("common.close")}
                </Button>
            </DialogActions>
        </CenteredPanel>
    );
}

const stageLabel = (
    stage: DebugStage,
    t: (key: string, options?: Record<string, unknown>) => string,
): string => {
    switch (stage.kind) {
        case "reconnecting":
            return t("settings.troubleshooting.stage.reconnecting");
        case "capturing":
            return t("settings.troubleshooting.stage.capturing");
        case "bundling":
            return t("settings.troubleshooting.stage.bundling");
        case "cancelling":
            return t("settings.troubleshooting.stage.cancelling");
        default:
            return "";
    }
};
