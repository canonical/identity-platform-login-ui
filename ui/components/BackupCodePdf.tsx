import React, { FC } from "react";
import {
  Document,
  Page,
  View,
  Text,
  StyleSheet,
  Image,
} from "@react-pdf/renderer";

interface Props {
  codes: string[];
}

const BackupCodePdf: FC<Props> = ({ codes }) => {
  return (
    <Document>
      <Page size="A4" style={styles.page}>
        <View>
          <Image
            src="./backup-codes-header.png"
            style={{ width: 530, marginBottom: 30 }}
          />
          <Text>These are your back up recovery codes.</Text>
          <Text style={{ marginBottom: 30 }}>
            Please keep them in a safe place!
          </Text>
          {codes.map((code, i) => (
            <Text
              key={i}
              style={{
                borderBottom: "1px solid #bbb",
                marginBottom: "10px",
                paddingBottom: "10px",
              }}
            >
              {i + 1}
              {"   "}
              {code}
            </Text>
          ))}
        </View>
      </Page>
    </Document>
  );
};

const styles = StyleSheet.create({
  page: {
    padding: 30,
    fontFamily: "Helvetica",
    fontSize: 14,
  },
});

export default BackupCodePdf;
