// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// Source codes whose connector accepts run parameters (domain.ConnectorConfigurable — today only
// osm.Connector.WithParameters, internal/congregationimport/adapters/connectors/osm). Kept in sync
// manually with the backend, same "add a source here when a new connector is registered" discipline
// page.tsx's own SOURCE_CODES already follows — there's no "which connectors take parameters"
// endpoint to fetch this from.
const PARAMETERIZED_SOURCES = new Set(["osm"]);

// Client component so the countryCodes sub-field can appear only when osm is actually selected —
// everything else on this page is a plain Server Component/Server Action, this is the one bit of
// genuine client-side interactivity the conditional field needs.
export function RunConnectorForm({
  sourceCodes,
  action,
}: {
  sourceCodes: string[];
  action: (formData: FormData) => void;
}) {
  const t = useTranslations("CongregationImportPage");
  const [sourceCode, setSourceCode] = useState(sourceCodes[0]);

  return (
    <form action={action} className="flex flex-wrap items-center gap-2">
      <Select name="sourceCode" value={sourceCode} onValueChange={setSourceCode}>
        <SelectTrigger size="sm" className="w-32">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {sourceCodes.map((code) => (
            <SelectItem key={code} value={code}>
              {code}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {PARAMETERIZED_SOURCES.has(sourceCode) && (
        <Input
          name="countryCodes"
          placeholder={t("runCountryCodesPlaceholder")}
          className="w-64"
        />
      )}
      <Button type="submit" size="sm">
        {t("runConnector")}
      </Button>
    </form>
  );
}
