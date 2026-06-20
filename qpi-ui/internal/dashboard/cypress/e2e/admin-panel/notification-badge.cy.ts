describe("Admin Panel — Notification Badge", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').type("admin@example.com");
    cy.get('input[type="password"]').type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");

    cy.contains("button", "Broadcast Announcement").click();
    cy.contains("button", "Broadcast Announcement")
      .should("have.class", "border-b-2");
  });

  it("shows a badge count > 0 after broadcast and returns to 0 after dismissing", () => {
    // Initially no badge
    cy.get('button svg.lucide-bell')
      .parent()
      .find("span.bg-red-500")
      .should("not.exist");

    // Broadcast an announcement
    cy.get('input[placeholder="QPU Maintenance Schedule"]')
      .type("System Update");
    cy.get('textarea[placeholder="Rigetti Aspen-9 will be offline for calibration tomorrow..."]')
      .type("A brief system update is scheduled.");
    cy.get("form").contains("button", "Broadcast Announcement").click();

    cy.contains("Announcement broadcasted successfully!").should("be.visible");

    // The bell should now show a badge with count > 0
    cy.get('button svg.lucide-bell')
      .parent()
      .find("span.bg-red-500")
      .should("be.visible")
      .and("contain", "1");

    // Open the dropdown and dismiss the notification
    cy.get('button svg.lucide-bell')
      .parent()
      .click();

    cy.get('button svg.lucide-x')
      .first()
      .parent()
      .click();

    // Badge should disappear (count returns to 0)
    cy.get('button svg.lucide-bell')
      .parent()
      .find("span.bg-red-500")
      .should("not.exist");
  });
});
