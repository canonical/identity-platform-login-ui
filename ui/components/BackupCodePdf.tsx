import React, { FC } from "react";
import { Document, Page, View, Text, StyleSheet } from "@react-pdf/renderer";

interface Props {
  codes: string[];
}

const BackupCodePdf: FC<Props> = ({ codes }) => {
  return (
    <Document>
      <Page size="A4" style={styles.page}>
        <View>
          <Text>
            These are your back up recovery codes. Please keep them in a safe
            place!
          </Text>
          {codes.map((code, i) => (
            <Text key={i}>{code}</Text>
          ))}
        </View>
      </Page>
    </Document>
  );
};

const styles = StyleSheet.create({
  page: {
    padding: 30,
  },
});

export default BackupCodePdf;
