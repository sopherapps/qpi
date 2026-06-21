describe("Admin Panel — Broadcast Announcement", () => {
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

  it("composes and broadcasts an announcement, then sees it in the bell dropdown", () => {
    const title = "Scheduled Maintenance";
    const description = "The QPU will be offline for calibration tomorrow.";

    // Fill the broadcast form
    cy.get('input[placeholder="QPU Maintenance Schedule"]')
      .type(title);
    cy.get('textarea[placeholder="Rigetti Aspen-9 will be offline for calibration tomorrow..."]')
      .type(description);
    
    // Fill datetime
    const pad = (n: number) => n.toString().padStart(2, "0");
    const start = new Date(Date.now() + 10 * 60000);
    const end = new Date(Date.now() + 20 * 60000);
    const startStr = `${start.getFullYear()}-${pad(start.getMonth() + 1)}-${pad(start.getDate())}T${pad(start.getHours())}:${pad(start.getMinutes())}`;
    const endStr = `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}T${pad(end.getHours())}:${pad(end.getMinutes())}`;

    cy.get('input[type="datetime-local"]').first().type(startStr);
    cy.get('input[type="datetime-local"]').last().type(endStr);

    // Submit the form
    cy.get("form").contains("button", "Broadcast Announcement").click();

    // Verify success message
    cy.contains("Announcement broadcasted successfully!").should("be.visible");

    // Open the notification bell dropdown
    cy.get('button svg.lucide-bell')
      .parent()
      .click();

    // The exact title should appear in the dropdown
    cy.contains("p", title).should("be.visible");
    cy.contains("p", description).should("be.visible");
  });
});
