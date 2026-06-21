describe("Admin Panel — Notification Badge", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");

    cy.contains("button", "Broadcast Announcement").click();
    cy.contains("button", "Broadcast Announcement")
      .should("have.class", "border-b-2");
  });

  it("shows a badge count > 0 after broadcast and returns to 0 after dismissing", () => {
    // Capture the initial badge count (may be >0 from prior test runs)
    cy.get('button svg.lucide-bell')
      .parent()
      .then(($btn) => {
        const $badge = $btn.find("span.bg-red-500");
        const initialCount = $badge.length > 0 ? parseInt($badge.text(), 10) : 0;

        // Broadcast an announcement
        cy.get('input[placeholder="QPU Maintenance Schedule"]')
          .clear().type("System Update");
        cy.get('textarea[placeholder="Rigetti Aspen-9 will be offline for calibration tomorrow..."]')
          .clear().type("A brief system update is scheduled.");
        
    // Fill datetime
    const pad = (n: number) => n.toString().padStart(2, "0");
    const start = new Date(Date.now() - 5 * 60000);
    const end = new Date(Date.now() + 20 * 60000);
    const startStr = `${start.getFullYear()}-${pad(start.getMonth() + 1)}-${pad(start.getDate())}T${pad(start.getHours())}:${pad(start.getMinutes())}`;
    const endStr = `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}T${pad(end.getHours())}:${pad(end.getMinutes())}`;

    cy.get('input[type="datetime-local"]').first().type(startStr);
    cy.get('input[type="datetime-local"]').last().type(endStr);

    cy.get("form").contains("button", "Broadcast Announcement").click();

        cy.contains("Announcement broadcasted successfully!").should("be.visible");

        // The bell should now show a badge with count = initial + 1
        cy.get('button svg.lucide-bell')
          .parent()
          .find("span.bg-red-500")
          .should("be.visible")
          .and("contain", (initialCount + 1).toString());

        // Open the dropdown and dismiss the most recent notification
        cy.get('button svg.lucide-bell')
          .parent()
          .click();

        cy.get('button svg.lucide-x')
          .first()
          .parent()
          .click();

        // Badge should return to the initial count
        if (initialCount === 0) {
          cy.get('button svg.lucide-bell')
            .parent()
            .find("span.bg-red-500")
            .should("not.exist");
        } else {
          cy.get('button svg.lucide-bell')
            .parent()
            .find("span.bg-red-500")
            .should("be.visible")
            .and("contain", initialCount.toString());
        }
      });
  });
});
