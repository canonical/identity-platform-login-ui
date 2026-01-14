// Snippet to fetch the latest recovery code from MailSlurper (Runs in Browser Console)
async function getRecoveryCode(email) {
    console.log(`Fetching code for ${email}...`);
    try {
        const res = await fetch('http://localhost:4437/mail');
        const data = await res.json();
        const mails = data.mailItems || [];
        
        // Filter for emails sent in the last 2 minutes to avoid stale codes
        const twoMinutesAgo = new Date(Date.now() - 2 * 60 * 1000);
        
        const myMails = mails
            .filter(m => m.toAddresses.includes(email))
            .filter(m => new Date(m.dateSent) > twoMinutesAgo)
            .sort((a, b) => new Date(b.dateSent) - new Date(a.dateSent));
            
        if (myMails.length === 0) return `NO_RECENT_EMAIL_FOUND_FOR_${email}`;
        
        const latest = myMails[0];
        console.log(`Found email: ${latest.subject} (${latest.dateSent})`);
        
        // Extract 6-8 digit code
        const match = latest.body.match(/\b\d{6,8}\b/);
        return match ? match[0] : "NO_CODE_IN_BODY";
    } catch (e) {
        return `ERROR: ${e.message}`;
    }
}
return await getRecoveryCode('TARGET_EMAIL_HERE');
