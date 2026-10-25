import type { ComponentType, SVGProps } from "react";
import { useTranslation } from "react-i18next";
import { Browser } from "@wailsio/runtime";
import { BookOpen } from "lucide-react";
// Sightwell fork: product lockup.
import netbirdFull from "@/assets/logos/sightwell-full.svg";
import netbirdFullLight from "@/assets/logos/sightwell-full-light.svg";
// Sightwell fork: community links come from i18n/brand.json (links.*).
import brand from "../../../../i18n/brand.json";

// Brand glyph from simpleicons.org (lucide deprecated its brand icons).
const SlackIcon = (props: SVGProps<SVGSVGElement>) => (
    <svg viewBox={"0 0 24 24"} fill={"currentColor"} {...props}>
        <path
            d={
                "M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52zM6.313 15.165a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.834 24a2.528 2.528 0 0 1-2.521-2.522v-6.313zM8.834 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.834 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.834zM8.834 6.313a2.528 2.528 0 0 1 2.521 2.521 2.528 2.528 0 0 1-2.521 2.521H2.522A2.528 2.528 0 0 1 0 8.834a2.528 2.528 0 0 1 2.522-2.521h6.312zM18.956 8.834a2.528 2.528 0 0 1 2.522-2.521A2.528 2.528 0 0 1 24 8.834a2.528 2.528 0 0 1-2.522 2.521h-2.522V8.834zM17.688 8.834a2.528 2.528 0 0 1-2.523 2.521 2.527 2.527 0 0 1-2.52-2.521V2.522A2.527 2.527 0 0 1 15.165 0a2.528 2.528 0 0 1 2.523 2.522v6.312zM15.165 18.956a2.528 2.528 0 0 1 2.523 2.522A2.528 2.528 0 0 1 15.165 24a2.527 2.527 0 0 1-2.52-2.522v-2.522h2.52zM15.165 17.688a2.527 2.527 0 0 1-2.52-2.523 2.526 2.526 0 0 1 2.52-2.52h6.313A2.527 2.527 0 0 1 24 15.165a2.528 2.528 0 0 1-2.522 2.523h-6.313z"
            }
        />
    </svg>
);
import { useSettings } from "@/contexts/SettingsContext.tsx";
import { useStatus } from "@/contexts/StatusContext.tsx";
import { UpdateVersionCard } from "@/modules/auto-update/UpdateVersionCard";
import { useAccentTrigger } from "@/modules/settings/SettingsAccent";

function openUrl(url: string) {
    Browser.OpenURL(url).catch(() => {
        window.open(url, "_blank");
    });
}

export function SettingsAbout() {
    const { t } = useTranslation();
    const { status } = useStatus();
    const { guiVersion } = useSettings();
    const daemonVersion = status?.daemonVersion ?? "—";

    const handleVersionClick = useAccentTrigger();

    const COMMUNITY_LINKS: {
        label: string;
        url: string;
        Icon: ComponentType<SVGProps<SVGSVGElement>>;
        iconClassName?: string;
    }[] = [
        // Sightwell fork: GitHub, Forum and Feedback (NetBird's) removed;
        // Slack and Documentation point at ours.
        {
            label: t("settings.about.community.slack"),
            url: brand.links.slack,
            Icon: SlackIcon,
            iconClassName: "h-3 w-3",
        },
        {
            label: t("settings.about.community.documentation"),
            url: brand.links.documentation,
            Icon: BookOpen,
        },
    ];

    // Sightwell fork: legal row from i18n/brand.json (legalLinks).
    // Empty hides the row (no Sightwell legal pages yet).
    const legalLinks: { key: string; url: string }[] = brand.legalLinks;
    const LEGAL_LINKS: { label: string; url: string }[] = legalLinks.map(({ key, url }) => ({
        label: t(`settings.about.links.${key}`),
        url,
    }));

    return (
        <div
            className={
                "mx-auto flex min-h-[calc(100vh-12rem)] max-w-2xl flex-col items-center justify-center gap-4"
            }
        >
            <img
                src={netbirdFullLight}
                alt={t("common.netbird")}
                className={"h-7 w-auto dark:hidden"}
            />
            <img
                src={netbirdFull}
                alt={t("common.netbird")}
                className={"hidden h-7 w-auto dark:block"}
            />
            <div className={"flex flex-col items-center gap-0.5 text-center"}>
                <button
                    type={"button"}
                    onClick={handleVersionClick}
                    className={
                        "cursor-text select-text bg-transparent text-sm font-semibold text-nb-gray-100 outline-none"
                    }
                >
                    {daemonVersion === "development" ? (
                        <span>
                            {t("settings.about.clientName")}{" "}
                            <span dir={"auto"} className={"font-mono text-yellow-400"}>
                                {t("settings.about.development")}
                            </span>
                        </span>
                    ) : (
                        t("settings.about.client", { version: daemonVersion })
                    )}
                </button>
                <p className={"cursor-text select-text text-sm font-medium text-nb-gray-250"}>
                    {guiVersion === "development" ? (
                        <span>
                            {t("settings.about.guiName")}{" "}
                            <span dir={"auto"} className={"font-mono text-yellow-400"}>
                                {t("settings.about.development")}
                            </span>
                        </span>
                    ) : (
                        t("settings.about.gui", { version: guiVersion })
                    )}
                </p>
            </div>

            <UpdateVersionCard />

            <p className={"mt-2 text-center text-sm text-nb-gray-300"}>
                {t("settings.about.copyright", { year: new Date().getFullYear() })}
            </p>
            <div
                className={"flex flex-wrap justify-center gap-x-4 gap-y-1 text-xs text-nb-gray-200"}
            >
                {COMMUNITY_LINKS.map(({ label, url, Icon, iconClassName }) => (
                    <button
                        key={url}
                        type={"button"}
                        tabIndex={0}
                        onClick={() => openUrl(url)}
                        className={
                            "inline-flex items-center gap-1.5 rounded-sm decoration-[0.5px] underline-offset-4 outline-none transition hover:text-nb-gray-100 hover:underline focus-visible:ring-2 focus-visible:ring-nb-gray-50/60 focus-visible:ring-offset-2 focus-visible:ring-offset-nb-gray-940"
                        }
                    >
                        <Icon aria-hidden={"true"} className={iconClassName ?? "h-3.5 w-3.5"} />
                        <span>{label}</span>
                    </button>
                ))}
            </div>
            {LEGAL_LINKS.length > 0 && (
                <div
                    className={
                        "flex flex-wrap justify-center gap-x-4 gap-y-1 text-xs text-nb-gray-200"
                    }
                >
                    {LEGAL_LINKS.map((link) => (
                        <button
                            key={link.url}
                            type={"button"}
                            tabIndex={0}
                            onClick={() => openUrl(link.url)}
                            className={
                                "rounded-sm decoration-[0.5px] underline-offset-4 outline-none transition hover:text-nb-gray-100 hover:underline focus-visible:ring-2 focus-visible:ring-nb-gray-50/60 focus-visible:ring-offset-2 focus-visible:ring-offset-nb-gray-940"
                            }
                        >
                            {link.label}
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
}
